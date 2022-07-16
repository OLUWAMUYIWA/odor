package utp

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const HeaderSize uint = 20

type PacketType uint8

const (
	Data  PacketType = iota // data payload
	Fin                     // sent as the end of a connection
	State                   // state of a packet, acked?
	Reset                   // forcibly ends a connection
	Syn                     // initiates a new connection with a remote socket
	Invalid
)

func packetType(n uint8) (PacketType, error) {
	if n < 0 || n > 5 {
		return Invalid, fmt.Errorf("Invalid package type: %d", n)
	}
	return PacketType(n), nil
}

func packetByte(p PacketType) uint8 {
	return uint8(p)
}

// only one extension type exists et
type ExtType uint8

const (
	None ExtType = iota
	SelectiveAck
	InvalidExt
)

func extType(n uint8) (ExtType, error) {
	if n == 0 {
		return None, nil
	} else if n == 1 {
		return SelectiveAck, nil
	} else {
		return InvalidExt, fmt.Errorf("Invalid ext type: %d", n)
	}
}

func extByte(e ExtType) uint8 {
	return uint8(e)
}

type Ext struct {
	data []byte
	ty   ExtType
}

func (e Ext) len() int {
	return len(e.data)
}

// etype returns the extension type of the extension
func (e Ext) etype() ExtType {
	return e.ty
}

func (e Ext) Iter() BitStream {
	return NewBitStream(e.data)
}

type PacketHeader struct {
	// type: first nibble, ver: second nibble
	typeVer      uint8
	extension    uint8
	connectionId uint16

	timestamp     uint32
	timestampDiff uint32

	wndSize uint32
	seqNr   uint16
	ackNr   uint16
}

// PckHdrFromByteSlice is used when we want to parse the header of a packet from the network
func PckHdrFromByteSlice(b []byte) (*PacketHeader, error) {
	if len(b) < int(HeaderSize) {
		return nil, fmt.Errorf("Packet length: %d is less than %d", len(b), HeaderSize)
	}

	// check version. version info is in the lower nibble
	if b[0]&0x0f != 1 {
		return nil, fmt.Errorf("Unsupported version")
	}
	// packet type is specified by the higher nibble
	if _, err := packetType(b[0] >> 4); err != nil {
		return nil, fmt.Errorf("Ivalid packet type")
	}

	return &PacketHeader{
		typeVer:       b[0],
		extension:     b[1],
		connectionId:  binary.BigEndian.Uint16(b[2:4]),
		timestamp:     binary.BigEndian.Uint32(b[4:8]),
		timestampDiff: binary.BigEndian.Uint32(b[8:12]),
		wndSize:       binary.BigEndian.Uint32(b[12:16]),
		seqNr:         binary.BigEndian.Uint16(b[16:18]),
		ackNr:         binary.BigEndian.Uint16(b[18:20]),
	}, nil
}

func PckHdrFromByteSliceUnchecked(b []byte) *PacketHeader {
	return &PacketHeader{
		typeVer:       b[0],
		extension:     b[1],
		connectionId:  binary.BigEndian.Uint16(b[2:4]),
		timestamp:     binary.BigEndian.Uint32(b[4:8]),
		timestampDiff: binary.BigEndian.Uint32(b[8:12]),
		wndSize:       binary.BigEndian.Uint32(b[12:16]),
		seqNr:         binary.BigEndian.Uint16(b[16:18]),
		ackNr:         binary.BigEndian.Uint16(b[18:20]),
	}
}

func NewPacketHeader() *PacketHeader {
	return new(PacketHeader)
}

func (h *PacketHeader) setType(t PacketType) {
	h.typeVer = (packetByte(t) << 4) | (h.typeVer & 0x0f)
}

func (h PacketHeader) getType() PacketType {
	t, _ := packetType(h.typeVer >> 4)
	return t
}

func (h PacketHeader) getVer() uint8 {
	return h.typeVer & 0x0f
}

func (h PacketHeader) getExtType() ExtType {
	ext, _ := extType(h.extension)
	return ext
}

func (h PacketHeader) asBytes() []byte {
	b := make([]byte, 20)
	b[0] = h.typeVer
	b[1] = h.extension
	binary.BigEndian.PutUint16(b[2:4], h.connectionId)
	binary.BigEndian.PutUint32(b[4:8], h.timestamp)
	binary.BigEndian.PutUint32(b[8:12], h.timestampDiff)
	binary.BigEndian.PutUint32(b[12:16], h.wndSize)
	binary.BigEndian.PutUint16(b[16:18], h.seqNr)
	binary.BigEndian.PutUint16(b[18:20], h.ackNr)
	return b
}

type Packet struct {
	data []byte
}

func (p Packet) inner() []byte {
	return p.data
}

func NewPacket() Packet {
	return Packet{
		data: PacketHeader{}.asBytes(),
	}
}

// PacketFromBytes checks if the header and the extensions are valid. It then creates a packet from the bytes
func PacketFromBytes(b []byte) (*Packet, error) {
	if _, err := PckHdrFromByteSlice(b); err != nil {
		return nil, err
	}
	if err := chectExts(b); err != nil {
		return nil, err
	}
	return &Packet{data: b}, nil
}

// NewPacketWithPayload creates a new data packet, with appropriate header and provided payload
func NewPacketWithPayload(b []byte) Packet {
	p := Packet{}
	pLen := int(HeaderSize) + len(b)
	data := make([]byte, pLen)
	hdr := NewPacketHeader()
	hdr.setType(Data)
	copy(data[0:20], hdr.asBytes())
	copy(data[20:], b)
	p.data = data
	return p
}

func (p *Packet) setType(t PacketType) {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	hdr.setType(t)
	copy(p.data[0:20], hdr.asBytes())
}

func (p Packet) getType() PacketType {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return hdr.getType()
}

func (p Packet) getVer() uint8 {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return hdr.getVer()
}

func (p Packet) getExtType() ExtType {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return hdr.getExtType()
}

func (p Packet) getExts() ExtIter {
	return NewExtIter(p)
}

// payload extracts the payload from the packet
func (p Packet) payload() []byte {
	index := int(HeaderSize)
	ext, _ := extType(p.data[1])
	// skip all extensions
	for index < len(p.data) && ext != None {
		len := p.data[index+1]
		ext, _ = extType(p.data[index])
		index += int(len) + 2
	}
	return p.data[index:]
}

func (p Packet) timestamp() TimeStamp {
	// we used the unchecked version of this method because we expect that a valid `Packet`  contains a valid `PacketHeader`
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	// this computer is little-endian, therefore we have to change the endianness
	return TimeStamp(invEndUint32(hdr.timestamp))
}

func (p *Packet) setTimestamp(t TimeStamp) {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	// invert the endianness before storing
	ts := invEndUint32(uint32(t))
	hdr.timestamp = ts
	copy(p.data[0:20], hdr.asBytes())
}

func (p Packet) timestampDiff() Delay {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return Delay(invEndUint32(hdr.timestampDiff))
}

func (p *Packet) setTimespanDiff(delay Delay) {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	hdr.timestamp = invEndUint32(uint32(delay))
	copy(p.data[:20], hdr.asBytes())
}

func (p Packet) getConnId() uint16 {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return invEndUint16(hdr.connectionId)
}

func (p Packet) getSeqNr() uint16 {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return invEndUint16(hdr.seqNr)
}

func (p Packet) getAckNr() uint16 {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return invEndUint16(hdr.ackNr)
}

func (p Packet) getWndSize() uint32 {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	return invEndUint32(hdr.wndSize)
}

func (p *Packet) setSeqNr(seqNr uint16) {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	hdr.seqNr = invEndUint16(seqNr)
	copy(p.data[:20], hdr.asBytes())
}

func (p *Packet) setAckNr(ackNr uint16) {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	hdr.ackNr = invEndUint16(ackNr)
	copy(p.data[:20], hdr.asBytes())
}

func (p *Packet) setConnID(connID uint16) {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	hdr.connectionId = invEndUint16(connID)
	copy(p.data[:20], hdr.asBytes())
}

func (p *Packet) setWndSize(wndSize uint32) {
	hdr := PckHdrFromByteSliceUnchecked(p.data[0:20])
	hdr.wndSize = invEndUint32(wndSize)
	copy(p.data[:20], hdr.asBytes())
}

// setSack sets Selective ACK field in packet header and adds appropriate data.
func (p *Packet) setSack(bv []byte) error {
	// The length of the SACK extension is expressed in bytes,
	// it ought be a multiple of 4 and at least 4.
	if len(bv) < 4 {
		return fmt.Errorf("length ought to be at least 4")
	}

	if len(bv)%4 != 0 {
		return fmt.Errorf("lenghth ought be divisible by 4")
	}

	// begin immediately after the header ends
	index := int(HeaderSize)
	ext, _ := extType(p.data[1])

	// wherever extenions end (become none), set the extension to be selective ack
	if ext == None {
		p.data[1] = byte(SelectiveAck)
	} else {
		for index < len(p.data) && ext != None {
			len := p.data[index+1]
			ext, _ = extType(p.data[index])
			if ext == None {
				p.data[index] = byte(SelectiveAck)
			}
			index += int(len) + 2
		}
	}
	left, right := p.data[:index], p.data[index:]
	left = append(left, byte(None))
	left = append(left, uint8(len(bv)))
	left = append(left, bv...)
	p.data = append(left, right...)

	return nil
}

func (p Packet) len() int {
	return len(p.data)
}

func (p Packet) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("type: %d\n", p.getType()))
	s.WriteString(
		fmt.Sprintf("version: %d\nextension: %d\nconnectionId: %d\ntimestamp: %d\ntimestampDiff: %d\nwndSize: %d\nseqNr: %d\nackNr: %d\n",
			p.getVer(), p.getExtType(), p.getConnId(), p.timestamp(), p.timestampDiff(), p.getWndSize(), p.getSeqNr(), p.getAckNr()))
	return s.String()
}

type ExtIter struct {
	b       []byte
	nextExt ExtType
	i       int // index
}

// comeback
func NewExtIter(p Packet) ExtIter {
	ex, _ := extType(p.data[1])
	return ExtIter{
		b:       p.data,
		nextExt: ex,
		i:       int(HeaderSize),
	}
}
func (e *ExtIter) next() (Ext, bool) { // the second return value indicates that it is done
	if e.nextExt == None {
		return Ext{}, false // done
	} else if e.i < len(e.b) {
		len := int(e.b[e.i+1])
		extStart := e.i + 2
		extEnd := extStart + len

		ext := Ext{
			ty:   e.nextExt,
			data: e.b[extStart:extEnd],
		}
		e.nextExt = ExtType(e.b[e.i])
		e.i += 2
		return ext, true
	} else {
		return Ext{}, false
	}
}

func chectExts(b []byte) error {
	if len(b) < int(HeaderSize) {
		return fmt.Errorf("Invalid Packet Length")
	}

	i := int(HeaderSize)
	extType, _ := extType(b[1])

	if len(b) == int(HeaderSize) && extType != None {
		return fmt.Errorf("Invali Extension Length")
	}

	for i < len(b) && extType != None {
		if len(b) < i+2 {
			return fmt.Errorf("Invalid Packet Length")
		}
		l := int(b[i+1])

		extStart := i + 2
		extEnd := extStart + l

		if l == 0 || l%4 != 0 || extEnd > len(b) {
			return fmt.Errorf("Invalid extension Length")
		}

		i += l + 2
	}

	if extType != None {
		return fmt.Errorf("Invalid Packet Length")
	}

	return nil
}
