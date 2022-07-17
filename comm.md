package utp

import (
	"errors"
	"fmt"
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
	/// The wrapped UDP socket
	socket *net.UDPAddr

	/// Remote peer
	connectedTo *SocketAddr

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

func NewSocketFromRaw(addr net.UDPAddr, remote SocketAddr) UtpSocket {
	sendID, rcvID := randSeqID()

	return UtpSocket{
		socket:                   &addr,
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

	utpSock := NewSocketFromRaw(*laddr, SocketAddr{ipAddr: net.IPAddr{IP: raddr.IP, Zone: raddr.Zone}, port: raddr.Port})

	p := NewPacket()
	p.setType(Syn)
	p.setConnID(utpSock.rcvrConnID)
	p.setSeqNr(utpSock.seqNr)

	l := 0
	buf := make([]byte, BUF_SIZE)
	udpConn, err := net.DialUDP("udp4", laddr, raddr)

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
		} else if errors.Is(err, os.ErrDeadlineExceeded) { // comeback to check for would block
			synTimeout += 2
			continue
		} else {
			return nil, err
		}

		// send packet

	}

	return nil, nil
}
