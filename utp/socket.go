package utp

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"
)

var IpV4Regex = regexp.MustCompile(`^[\d{2}]\.[\d{2}]\.[\d{2}]\.[\d{2}]:[\d{2}$`)

// For simplicity's sake, let us assume no packet will ever exceed the Ethernet maximum transfer unit of 1500 bytes.
const BUF_SIZE uint = 1500
const GAIN float64 = 1.0
const ALLOWED_INCREASE uint32 = 1
const TARGET float64 = 100_000.0 // 100 milliseconds
const MSS uint32 = 1400
const MIN_CWND uint32 = 2
const INIT_CWND uint32 = 2
const INITIAL_CONGESTION_TIMEOUT uint64 = 1000 // one second
const MIN_CONGESTION_TIMEOUT uint64 = 500      // 500 ms
const MAX_CONGESTION_TIMEOUT uint64 = 60_000   // one minute
const BASE_HISTORY uint = 10                   // base delays history size
const MAX_SYN_RETRIES uint32 = 5               // maximum connection retries
const MAX_RETRANSMISSION_RETRIES uint32 = 5    // maximum retransmission retries
const WINDOW_SIZE uint32 = 1024 * 1024         // local receive window size
// Maximum time (in microseconds) to wait for incoming packets when the send window is full
const PRE_SEND_TIMEOUT uint32 = 500_000

const MAX_BASE_DELAY_AGE Delay = 60_000_000

type SocketState uint8

const (
	New SocketState = iota
	Connected
	SynSent
	FinSent
	ResetReceived
	Closed
)

type DelayDifferenceSample struct {
	receivedAt TimeStamp
	difference Delay
}

type SocketAddr struct {
	ipAddr net.IPAddr
	port   int
}

func (s SocketAddr) String() string {
	return net.JoinHostPort(s.ipAddr.String(), strconv.Itoa(s.port))
}

type UtpSocket struct {
	logger *log.Logger

	// the udp conn
	conn *net.UDPConn

	/// The wrapped UDP socket
	socket *net.UDPAddr

	/// Remote peer
	connectedTo *net.UDPAddr

	/// Sender connection identifier
	senderConnID uint16

	/// Receiver connection identifier
	rcvrConnID uint16

	/// Sequence number for the next packet
	seqNr uint16

	/// Sequence number of the latest acknowledged packet sent by the remote peer
	ackNr uint16

	/// Socket state
	state SocketState

	/// Received but not acknowledged packets
	incomingBuff []Packet

	/// Sent but not yet acknowledged packets
	sendWdw []Packet

	/// Packets not yet sent
	unsentQueue []Packet

	/// How many ACKs did the socket receive for packet with sequence number equal to `ack_nr`
	dupAckCount uint32

	/// Sequence number of the latest packet the remote peer acknowledged
	lastAcked uint16

	/// Timestamp of the latest packet the remote peer acknowledged
	lastAckedTimestamp TimeStamp

	/// Sequence number of the last packet removed from the incoming buffer
	lastDropped uint16

	/// Round-trip time to remote peer
	rtt int32

	/// Variance of the round-trip time to the remote peer
	rttVariance int32

	/// Data from the latest packet not yet returned in `recv_from`
	pendingData []uint8

	/// Bytes in flight
	currWdw uint32

	/// Window size of the remote peer
	remoteWndSize uint32

	/// Rolling window of packet delay to remote peer
	baseDelays []Delay

	/// Rolling window of the difference between sending a packet and receiving its acknowledgement
	currentDelays []DelayDifferenceSample

	/// Difference between timestamp of the latest packet received and time of reception
	theirDelay Delay

	/// Start of the current minute for sampling purposes
	lastRollover TimeStamp

	/// Current congestion timeout in milliseconds
	congestionTimeout uint64

	/// Congestion window in bytes
	cwnd uint32

	/// Maximum retransmission retries
	maxRetransmissionRetries uint32
}

func NewSocketFromRaw(addr *net.UDPAddr, remote *net.UDPAddr, conn *net.UDPConn) UtpSocket {
	sendID, rcvID := randSeqID()

	return UtpSocket{
		logger:                   log.New(os.Stdout, "utp", log.LUTC),
		conn:                     conn,
		socket:                   addr,
		connectedTo:              remote,
		senderConnID:             sendID,
		rcvrConnID:               rcvID,
		seqNr:                    1,
		ackNr:                    0,
		state:                    New,
		incomingBuff:             []Packet{},
		sendWdw:                  []Packet{},
		unsentQueue:              []Packet{},
		dupAckCount:              0,
		lastAcked:                0,
		lastAckedTimestamp:       TimeStamp(0),
		lastDropped:              0,
		rtt:                      0,
		rttVariance:              0,
		pendingData:              []uint8{},
		currWdw:                  0,
		remoteWndSize:            0,
		currentDelays:            []DelayDifferenceSample{},
		baseDelays:               []Delay{},
		theirDelay:               Delay(0),
		lastRollover:             TimeStamp(0),
		congestionTimeout:        INITIAL_CONGESTION_TIMEOUT,
		cwnd:                     INIT_CWND * MSS,
		maxRetransmissionRetries: MAX_RETRANSMISSION_RETRIES,
	}
}

func (u UtpSocket) localAddr() string {
	return u.socket.String()
}

func (u UtpSocket) peerAddr() (string, error) {
	if u.state == Connected || u.state == FinSent {
		return u.connectedTo.String(), nil
	}

	return "", fmt.Errorf("Not Connected")
}

func connect(addr SocketAddr) (*UtpSocket, error) {
	raddr, err := net.ResolveUDPAddr("udp", addr.String())

	if err != nil {
		return nil, err
	}
	var lAddr string
	if IpV4Regex.MatchString(raddr.String()) {
		lAddr = "0.0.0.0:0"
	} else {
		lAddr = "[::]:0"
	}
	laddr, err := net.ResolveUDPAddr("udp", lAddr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.DialUDP("udp4", laddr, raddr)

	// utpSock := NewSocketFromRaw(*laddr, SocketAddr{ipAddr: net.IPAddr{IP: raddr.IP, Zone: raddr.Zone}, port: raddr.Port})
	utpSock := NewSocketFromRaw(laddr, raddr, udpConn)

	p := NewPacket()
	p.setType(Syn)
	p.setConnID(utpSock.rcvrConnID)
	p.setSeqNr(utpSock.seqNr)

	l := 0
	buf := make([]byte, BUF_SIZE)

	synTimeout := utpSock.congestionTimeout

	for i := 0; i < int(MAX_SYN_RETRIES); i++ {
		p.setTimestamp(nowMicroSecs())

		if _, err := udpConn.Write(p.asBytes()); err != nil {
			return nil, err
		}

		utpSock.state = SynSent

		if err := udpConn.SetReadDeadline(time.Now().Add(time.Duration(synTimeout))); err != nil {
			return nil, err
		}

		if n, addr, err := udpConn.ReadFromUDP(buf); err == nil {
			utpSock.connectedTo = addr
			l = n
			break
		} else if errors.Is(err, os.ErrDeadlineExceeded) { // comeback to check for would block
			synTimeout += 2
			continue
		} else {
			return nil, err
		}
	}

	// comeback : reset read deadline: do i need to do this?
	udpConn.SetReadDeadline(time.Time{})

	address := utpSock.connectedTo
	packet, err := PacketFromBytes(buf[:l])
	if err != nil {
		return nil, err
	}
	if _, err := utpSock.handlePacket(packet, address); err != nil {
		return nil, err
	}

	return &utpSock, nil
}

func (u *UtpSocket) Close() error {
	if u.state == Closed || u.state == New || u.state == SynSent {
		return nil
	}
	if err := u.Flush(); err != nil {
		return err
	}

	p := NewPacket()
	p.setConnID(u.senderConnID)
	p.setSeqNr(u.seqNr)
	p.setAckNr(u.ackNr)
	p.setTimestamp(nowMicroSecs())
	p.setType(Fin)

	if _, err := u.conn.Write(p.asBytes()); err != nil {
		return err
	}

	u.state = FinSent

	b := make([]byte, BUF_SIZE)
	for u.state != Closed {
		if _, _, err := u.recv(b); err != nil {
			return err
		}
	}

	return nil
}

func (u *UtpSocket) RecvFrom(b []byte) (int, *net.UDPAddr, error) {
	read := u.FlushIncomingBuffer(b)
	if read > 0 {
		return read, u.connectedTo, nil
	}

	if u.state == ResetReceived {
		return 0, nil, fmt.Errorf("Connection reset")
	}

	for {
		if u.state == Closed {
			return 0, u.connectedTo, nil
		}

		n, addr, err := u.recv(b)
		if err != nil {
			return 0, nil, err
		}

		if n == 0 {
			continue
		} else {
			return n, addr, nil
		}
	}
}

func (u *UtpSocket) recv(buf []byte) (int, *net.UDPAddr, error) {
	b := make([]byte, BUF_SIZE+HeaderSize)
	start := time.Now()
	retries := 0
	var nRead int
	var rmtSource *net.UDPAddr
	var err error
	for {
		if retries >= int(u.maxRetransmissionRetries) {
			u.state = Closed
			return 0, nil, os.ErrDeadlineExceeded
		}

		// try to set read deadline
		if u.state != New {
			u.conn.SetReadDeadline(time.Now().Add(time.Duration(time.Duration(u.congestionTimeout).Milliseconds()))) // convert to milisecond from microsecond
		} else {
			u.conn.SetReadDeadline(time.Time{})
		}

		nRead, rmtSource, err = u.conn.ReadFromUDP(b)
		if err == nil {
			break
		}
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) || os.IsTimeout(err) { // comeback: i suppose os.Timeout() hecks for E_WOULDBLOCK
				if err := u.handleRecieveTimeout(); err != nil {
					return 0, nil, err
				}
			} else {
				return 0, nil, err
			}
		}

		elapsed := time.Since(start).Milliseconds()
		u.logger.Printf("Elapsed: %d milliseonds\n", elapsed)
		retries += 1
	}

	packet, err := PacketFromBytes(b[:nRead])
	if err != nil {
		u.logger.Printf("Ignoring invalid packet: %s\n", err)
		return 0, u.connectedTo, nil
	}

	u.logger.Printf("received: %s", *packet)
	pkt, err := u.handlePacket(packet, rmtSource)
	if err != nil {
		return 0, nil, err
	}

	if pkt != nil {
		pkt.setWndSize(WINDOW_SIZE)
		if _, err = u.conn.WriteToUDP(pkt.asBytes(), rmtSource); err != nil {
			return 0, nil, err
		}
		u.logger.Printf("sent: %s", pkt)
	}

	read := u.FlushIncomingBuffer(buf)

	return read, rmtSource, nil
}

func (u *UtpSocket) handleRecieveTimeout() error {
	u.congestionTimeout += 2
	u.cwnd = MSS
	// There are three possible cases here:
	//
	// - If the socket is sending and waiting for acknowledgements (the send window is
	//   not empty), resend the first unacknowledged packet;
	//
	// - If the socket is not sending and it hasn't sent a FIN yet, then it's waiting
	//   for incoming packets: send a fast resend request;
	//
	// - If the socket sent a FIN previously, resend it.

	if len(u.sendWdw) == 0 {
		// The socket is trying to close, all sent packets were acknowledged, and it has
		// already sent a FIN: resend it.
		pkt := NewPacket()
		pkt.setConnID(u.senderConnID)
		pkt.setSeqNr(u.seqNr)
		pkt.setAckNr(u.ackNr)
		pkt.setTimestamp(nowMicroSecs())
		pkt.setType(Fin)

		if _, err := u.conn.WriteToUDP(pkt.asBytes(), u.connectedTo); err != nil {
			return err
		}
		u.logger.Printf("resent FIN: %s\n", pkt)
	} else if u.state == New {
		// The socket is waiting for incoming packets but the remote peer is silent:
		// send a fast resend request.
		u.logger.Println("sending fast resend request")
		u.sendFastRsndReq()
	} else {
		packet := u.sendWdw[0]
		packet.setTimestamp(nowMicroSecs())
		if _, err := u.conn.WriteToUDP(packet.asBytes(), u.connectedTo); err != nil {
			return err
		}
	}
	return nil
}

func (u *UtpSocket) sendFastRsndReq() {
	for i := 0; i < 3; i++ {
		pkt := NewPacket()
		pkt.setType(State)
		pkt.setTimestamp(nowMicroSecs())
		pkt.setTimespanDiff(u.theirDelay)
		pkt.setConnID(u.senderConnID)
		pkt.setAckNr(u.seqNr)
		pkt.setAckNr(u.ackNr)
		u.conn.WriteToUDP(pkt.asBytes(), u.connectedTo)
	}
}

func (u *UtpSocket) FlushIncomingBuffer(b []byte) int {
	if len(u.pendingData) != 0 {
		nFlushed := copy(b, u.pendingData)
		if nFlushed == len(u.pendingData) {
			u.pendingData = []byte{}
			u.advIncomingBuf()
		} else {
			u.pendingData = u.pendingData[nFlushed:]
		}

		return nFlushed
	}

	if len(u.incomingBuff) != 0 && ((u.ackNr == u.incomingBuff[0].getSeqNr()) || (u.ackNr+1 == u.incomingBuff[0].getSeqNr())) {
		nFlushed := copy(b, u.incomingBuff[0].payload())
		if nFlushed == len(u.incomingBuff[0].payload()) {
			u.advIncomingBuf()
		} else {
			u.pendingData = u.incomingBuff[0].payload()[nFlushed:]
		}

		return nFlushed
	}

	return 0
}

func (u *UtpSocket) advIncomingBuf() {

}

func (c *UtpSocket) Flush() error {
	buf := make([]byte, BUF_SIZE)
	for len(c.sendWdw) != 0 {
		if _, _, err := c.recv(buf); err != nil {
			return err
		}
	}
	return nil
}
func (u *UtpSocket) handlePacket(p *Packet, addr *net.UDPAddr) (*Packet, error) {
	// Acknowledge only if the packet strictly follows the previous one
	if p.getSeqNr()-p.getAckNr() == 1 {
		u.ackNr = p.getSeqNr()
	}

	// Reset connection if connection id doesn't match and this isn't a SYN
	if p.getType() != Syn && u.state != SynSent && !(p.getConnId() == u.senderConnID || p.getConnId() == u.rcvrConnID) {
		return u.prepareReply(p, Reset), nil
	}

	// Update remote window size
	u.remoteWndSize = p.getWndSize()
	u.logger.Printf("UTP remote_wnd_size: %d", u.remoteWndSize)

	// Update remote peer's delay between them sending the packet and us receiving it
	u.theirDelay = absDiff(nowMicroSecs(), p.timestamp())
	u.logger.Printf("their_delay: %d\n", u.theirDelay)

	state, ty := u.state, p.getType()
	if state == New && ty == Syn {
		u.connectedTo = addr
		u.ackNr = p.getSeqNr()
		u.seqNr = uint16(rand.Int31())
		u.rcvrConnID = p.getConnId() + 1
		u.senderConnID = p.getConnId()
		u.state = Connected
		u.lastDropped = u.ackNr

		return u.prepareReply(p, State), nil
	} else if ty == Syn {
		return u.prepareReply(p, Reset), nil
	} else if state == SynSent && ty == State {
		u.connectedTo = addr
		u.ackNr = p.getSeqNr()
		u.seqNr += 1
		u.state = Connected
		u.lastAcked = p.getAckNr()
		u.lastAckedTimestamp = nowMicroSecs()
		return nil, nil // Okay(None)
	} else if state == SynSent {
		return nil, fmt.Errorf("Invalid Reply")
	} else if (state == Connected && ty == Data) || (state == FinSent && ty == Data) {
		return u.handleDataPacket(p), nil
	} else if state == Connected && ty == State {
		u.handleStatePacket(p)
		return nil, nil
	} else if (state == Connected && ty == Fin) || (state == FinSent && ty == Fin) {
		if p.getAckNr() < u.seqNr {
			u.logger.Println("FIN received but there are missing acknowledgements for sent packets")
		}
		reply := u.prepareReply(p, State)

		if p.getSeqNr()-p.getAckNr() > 1 {
			// Set SACK extension payload if the packet is not in order
			sack := u.buildSelectiveAck()
			if len(sack) != 0 {
				reply.setSack(sack)
			}
		}
		u.state = Closed
		return reply, nil
	} else if state == Closed && ty == Fin {
		return u.prepareReply(p, State), nil
	} else if state == FinSent && ty == State {
		if p.getAckNr() == u.seqNr {
			u.state = Closed
		} else {
			u.handleStatePacket(p)
		}
		return nil, nil
	} else if ty == Reset {
		u.state = ResetReceived
		return nil, fmt.Errorf("Connection Reset")
	} else {
		u.logger.Printf("Unimplemented handling for (%d, %d)\n", state, ty)
		return nil, fmt.Errorf("Unimplemented handling for (%d, %d)\n", state, ty)
	}
}

func (u *UtpSocket) prepareReply(original *Packet, t PacketType) *Packet {
	ret := NewPacket()
	ret.setType(t)
	currT := nowMicroSecs()
	otherT := original.timestamp()
	timeDiff := absDiff(currT, otherT)
	ret.setTimestamp(currT)
	ret.setTimespanDiff(timeDiff)
	ret.setConnID(u.senderConnID)
	ret.setSeqNr(u.seqNr)
	ret.setAckNr(u.ackNr)

	return &ret
}

func (u *UtpSocket) handleDataPacket(p *Packet) *Packet {
	var ty PacketType
	if u.state == FinSent {
		ty = Fin
	} else {
		ty = State
	}

	reply := u.prepareReply(p, ty)
	if p.getSeqNr()-u.ackNr > 1 {
		u.logger.Printf("current ack_nr %d is behind received packet seq_nr (%d)\n", u.ackNr, p.getSeqNr())
		// Set SACK extension payload if the packet is not in order
		sack := u.buildSelectiveAck()
		if len(sack) != 0 {
			reply.setSack(sack)
		}
	}

	return reply
}

func (u *UtpSocket) handleStatePacket(p *Packet) {
	if p.getAckNr() == u.lastAcked {
		u.dupAckCount += 1
	} else {
		u.lastAcked = p.getAckNr()
		u.lastAckedTimestamp = nowMicroSecs()
		u.dupAckCount = 1
	}

	// Update congestion window size
	for index, pkt := range u.sendWdw {
		if p.getAckNr() == pkt.getSeqNr() {
			// Calculate the sum of the size of every packet implicitly and explicitly acknowledged
			// by the inbound packet (i.e., every packet whose sequence number precedes the inbound
			// packet's acknowledgement number, plus the packet whose sequence number matches)

			var bytesNewlyAcked int
			for i := 0; i <= index; i++ {
				bytesNewlyAcked += u.sendWdw[i].len()
			}
			// Update base and current delay
			now := nowMicroSecs()
			ourDelay := now - u.sendWdw[index].timestamp()
			u.logger.Printf("our delay: %d\n", ourDelay)
			u.updateBaseDelay(Delay(ourDelay), now)
			u.updateCurrDelay(Delay(ourDelay), now)

			offTarget := TARGET - float64(uint32(u.queuingDelay()))/TARGET
			u.updateCongestionWdw(offTarget, uint32(bytesNewlyAcked))
			rtt := uint32(ourDelay-TimeStamp(u.queuingDelay())) / 1000 // in milli
			u.updateCongestionTimeOut(int32(rtt))
			break
		}
	}

	var pktLossDetected bool
	if len(u.sendWdw) != 0 && u.dupAckCount == 3 {
		pktLossDetected = true
	}

	// Process extensions, if any
	extIter := p.getExts()
	ext, ok := extIter.next()
	for ; ok; ext, ok = extIter.next() {
		if ext.etype() == SelectiveAck {
			// If three or more packets are acknowledged past the implicit missing one,
			// assume it was lost.
			bitStr := NewBitStream(ext.data)
			if bitStr.CountOnes() >= 3 {
				u.resendLostPacket(p.getAckNr() + 1)
				pktLossDetected = true
			}
			if len(u.sendWdw) != 0 {
				lastSeqNr := u.sendWdw[len(u.sendWdw)-1].getSeqNr()
				lostPackets := []uint16{}
				rcvd, err := bitStr.Next()
				for i := 0; err == nil; rcvd, err = bitStr.Next() {
					if !rcvd {
						seqNr := p.getAckNr() + 2 + uint16(i)
						if seqNr < lastSeqNr {
							lostPackets = append(lostPackets, seqNr)
						}
					}
					i += 1
				}

				for _, seqNr := range lostPackets {
					u.logger.Printf("SACK: packet %d lost\n", seqNr)
					u.resendLostPacket(seqNr)
					pktLossDetected = true
				}
			} else {
				u.logger.Printf("Unknown extension %d, ignoring", ext.etype())
			}
		}
	}

	// Three duplicate ACKs mean a fast resend request. Resend the first unacknowledged packet
	// if the incoming packet doesn't have a SACK extension. If it does, the lost packets were
	// already resent.
	ext, ok = extIter.next()
	var anySelectiveAck bool
	for ; ok; ext, ok = extIter.next() {
		if ext.etype() == SelectiveAck {
			anySelectiveAck = true
			break
		}
	}
	if len(u.sendWdw) != 0 && u.dupAckCount != 3 && !anySelectiveAck {
		u.resendLostPacket(p.getAckNr() + 1)
	}

	// Packet lost, halve the congestion window
	if pktLossDetected {
		u.logger.Println("packet loss detected, halving congestion window")
		u.cwnd = max(u.cwnd/2, MIN_CWND*MSS)
	}

	// Success, advance send window
	u.advanceSendWindow()
}

// advanceSendWindow forgets sent packets that were acknowledged by the remote peer.
func (u *UtpSocket) advanceSendWindow() {
	// The reason I'm not removing the first element in a loop while its sequence number is
	// smaller than `last_acked` is because of wrapping sequence numbers, which would create the
	// sequence [..., 65534, 65535, 0, 1, ...]. If `last_acked` is smaller than the first
	// packet's sequence number because of wraparound (for instance, 1), no packets would be
	// removed, as the condition `seq_nr < last_acked` would fail immediately.
	//
	// On the other hand, I can't keep removing the first packet in a loop until its sequence
	// number matches `last_acked` because it might never match, and in that case no packets
	// should be removed.
	for pos, p := range u.sendWdw {
		if p.getSeqNr() == u.lastAcked {
			for i := 0; i <= pos; i++ {
				packet := u.sendWdw[0]
				u.sendWdw = u.sendWdw[1:]
				u.currWdw -= uint32(packet.len())
			}
			break
		}
	}
	u.logger.Printf("self.curr_window: %v\n", u.currWdw)
}

// sendPacket sends one packet
func (u *UtpSocket) sendPacket(p *Packet) error {
	u.logger.Printf("current window: %d\n", len(u.sendWdw))
	maxInFlight := min(u.cwnd, u.remoteWndSize)
	maxInFlight = max(MIN_CWND*MSS, maxInFlight)

	now := nowMicroSecs()
	// Wait until enough in-flight packets are acknowledged for rate control purposes, but don't
	// wait more than 500 ms (PRE_SEND_TIMEOUT) before sending the packet.
	for u.currWdw >= maxInFlight && nowMicroSecs()-now < TimeStamp(PRE_SEND_TIMEOUT) {
		u.logger.Printf("self.curr_window: %d\n", u.currWdw)
		u.logger.Printf("max_inflight: %d\n", maxInFlight)
		u.logger.Printf("u.duplicate_ack_count: %d\n", u.dupAckCount)
		u.logger.Printf("now_microseconds() - now = %d\n", nowMicroSecs()-now)
		buf := make([]byte, BUF_SIZE)
		if _, _, err := u.recv(buf); err != nil {
			return err
		}
	}
	u.logger.Printf("out: now_microseconds() - now = %d\n", nowMicroSecs()-now)
	// Check if it still makes sense to send packet, as we might be trying to resend a lost
	// packet acknowledged in the receive loop above.
	// If there were no wrapping around of sequence numbers, we'd simply check if the packet's
	// sequence number is greater than `last_acked`.

	// comeback to implement wrapping sub here
	distA := p.getSeqNr() - u.lastAcked
	distB := u.lastAcked - p.getSeqNr()
	if distA > distB {
		u.logger.Println("Packet already acknowledged, skipping...")
		return nil
	}

	p.setTimestamp(nowMicroSecs())
	p.setTimespanDiff(u.theirDelay)
	if _, err := u.conn.WriteToUDP(p.asBytes(), u.connectedTo); err != nil {
		return err
	}
	u.logger.Printf("Sent: %s\n", *p)
	return nil
}
func (u *UtpSocket) resendLostPacket(lostPktNr uint16) {
	u.logger.Printf("---> resend_lost_packet(%d) <---\n", lostPktNr)
	var found bool
	for pos, p := range u.sendWdw {
		if p.getSeqNr() == lostPktNr {
			found = true
			u.logger.Printf("u.send_window.len(): %d\n", len(u.sendWdw))
			u.logger.Printf("Position: %d\n", pos)
			u.sendPacket(&u.sendWdw[pos])
			// We intentionally don't increase `curr_window` because otherwise a packet's length
			// would be counted more than once
			break
		}
	}

	if !found {
		u.logger.Printf("Packet %d not found\n", lostPktNr)
	}

	u.logger.Println("---> END resend_lost_packet <---")
}

type diff struct {
	byt, bit int
}

// buildSelectivAck builds the selective acknowledgement extension data for usage in packets.
func (u *UtpSocket) buildSelectiveAck() []byte {
	stashed := []diff{}
	for _, p := range u.incomingBuff {
		if p.getSeqNr() > u.seqNr+1 {
			dif := int(p.getSeqNr() - u.ackNr - 2)
			byt, bit := dif/8, dif%8
			stashed = append(stashed, diff{byt: byt, bit: bit})
		}
	}
	sack := []byte{}
	for _, d := range stashed {
		// Make sure the amount of elements in the SACK vector is a
		// multiple of 4 and enough to represent the lost packets
		for d.byt > len(sack) || len(sack)%4 == 0 {
			sack = append(sack, 0)
		}
		sack[d.byt] |= 1 << d.bit
	}
	return sack
}

// updateBaseDelay inserts a new sample in the base delay list.
// The base delay list contains at most `BASE_HISTORY` samples, each sample is the minimum
// measured over a period of a minute (MAX_BASE_DELAY_AGE).
func (u *UtpSocket) updateBaseDelay(baseDelay Delay, now TimeStamp) {
	if len(u.baseDelays) == 0 || int64(now)-int64(u.lastRollover) > int64(MAX_BASE_DELAY_AGE) {
		// update last rollover
		u.lastRollover = now

		// drop oldest dample if need be
		if len(u.baseDelays) == int(BASE_HISTORY) {
			if len(u.baseDelays) != 0 {
				u.baseDelays = u.baseDelays[1:]
			}
		}
		// insert new sample
		u.baseDelays = append(u.baseDelays, baseDelay)
	} else {
		// Replace sample for the current minute if the delay is lower
		lastIdx := len(u.baseDelays) - 1
		if baseDelay < u.baseDelays[lastIdx] {
			u.baseDelays[lastIdx] = baseDelay
		}

	}
}

// updateCurrDelay inserts a new sample in the current delay list after removing samples older than one RTT, as
// specified in RFC6817.
func (u *UtpSocket) updateCurrDelay(d Delay, now TimeStamp) {
	// Remove samples more than one RTT old
	rtt := Delay(int64(u.rtt) * 100)
	for len(u.currentDelays) != 0 && int64(now)-int64(u.currentDelays[0].receivedAt) > int64(rtt) {
		u.currentDelays = u.currentDelays[1:]
	}

	// insert new measurement
	u.currentDelays = append(u.currentDelays, DelayDifferenceSample{
		receivedAt: now,
		difference: d,
	})

}

// Calculates the new congestion window size, increasing it or decreasing it.
//
// This is the core of uTP, the [LEDBAT][ledbat_rfc] congestion algorithm. It depends on
// estimating the queuing delay between the two peers, and adjusting the congestion window
// accordingly.
//
// `off_target` is a normalized value representing the difference between the current queuing
// delay and a fixed target delay (`TARGET`). `off_target` ranges between -1.0 and 1.0. A
// positive value makes the congestion window increase, while a negative value makes the
// congestion window decrease.
//
// `bytes_newly_acked` is the number of bytes acknowledged by an inbound `State` packet. It may
// be the size of the packet explicitly acknowledged by the inbound packet (i.e., with sequence
// number equal to the inbound packet's acknowledgement number), or every packet implicitly
// acknowledged (every packet with sequence number between the previous inbound `State`
// packet's acknowledgement number and the current inbound `State` packet's acknowledgement
// number).
//
//[ledbat_rfc]: https://tools.ietf.org/html/rfc6817
func (u *UtpSocket) updateCongestionWdw(offTarget float64, bytesNewlyAcked uint32) {
	flightsize := u.currWdw

	cwndIncr := GAIN * offTarget * float64(bytesNewlyAcked) * float64(MSS)
	cwndIncr = cwndIncr / float64(u.cwnd)
	u.logger.Printf("cwnd increase: %f\n", cwndIncr)

	u.cwnd = uint32(float64(u.cwnd) + cwndIncr)
	maxAllowedCwnd := flightsize + ALLOWED_INCREASE*MSS
	u.cwnd = min(u.cwnd, maxAllowedCwnd)
	u.cwnd = max(u.cwnd, MIN_CWND*MSS)

	u.logger.Printf("cwnd: %d\n", u.cwnd)
	u.logger.Printf("max_allowed_cwnd: %d\n", maxAllowedCwnd)
}

func (u *UtpSocket) updateCongestionTimeOut(currDelay int32) {
	delta := u.rtt - currDelay
	u.rtt += (abs(delta) - u.rttVariance) / 4
	u.congestionTimeout = max(uint64(u.rtt+u.rttVariance*4), MIN_CONGESTION_TIMEOUT)
	u.congestionTimeout = min(u.congestionTimeout, MIN_CONGESTION_TIMEOUT)

	u.logger.Printf("current_delay: %d\n", currDelay)
	u.logger.Printf("delta: %d\n", delta)
	u.logger.Printf("u.rtt_variance: %d\n", u.rttVariance)
	u.logger.Printf("u.rtt: %d\n", u.rtt)
	u.logger.Printf("u.congestion_timeout: %d\n", u.congestionTimeout)
}

func (u *UtpSocket) queuingDelay() Delay {
	filteredCurrentDelay := u.filteredCurrentDelay()
	minBaseDelay := u.minBaseDelay()
	queuingDelay := filteredCurrentDelay - minBaseDelay

	u.logger.Printf("filtered_current_delay:%d\n", filteredCurrentDelay)
	u.logger.Printf("min_base_delay:%d\n", minBaseDelay)
	u.logger.Printf("queuing_delay:%d\n", queuingDelay)

	return queuingDelay
}

// calculates the filtered current delay in the current window.
// The current delay is calculated through application of the exponential
// weighted moving average filter with smoothing factor 0.333 over the
// current delays in the current window.
func (u UtpSocket) filteredCurrentDelay() Delay {
	_ = u.currentDelays
	return 0
}

func (u UtpSocket) minBaseDelay() Delay {
	return 0
}
