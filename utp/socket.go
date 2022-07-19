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
	received_at TimeStamp
	difference  Delay
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

	// var pktLossDetected bool
	// if len(u.sendWdw) != 0 && u.dupAckCount == 3 {
	// 	pktLossDetected = true
	// }

	// // Process extensions, if any
	// extIter := p.getExts()

}

func (u *UtpSocket) buildSelectiveAck() []byte {
	return nil
}

func (u *UtpSocket) updateBaseDelay(d Delay, t TimeStamp) {

}

func (u *UtpSocket) updateCurrDelay(d Delay, t TimeStamp) {

}

func (u *UtpSocket) updateCongestionWdw(offTarget float64, bytesNewlyAcked uint32) {

}

func (u *UtpSocket) updateCongestionTimeOut(currDelay int32) {

}

func (u *UtpSocket) queuingDelay() Delay {
	return 0
}
