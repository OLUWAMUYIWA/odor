package utp

import (
	"encoding/binary"
	"fmt"
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
	if n == uint8(Data) {
		return Data, nil
	} else if n == uint8(Fin) {
		return Fin, nil
	} else if n == uint8(State) {
		return State, nil
	} else if n == uint8(Reset) {
		return Reset, nil
	} else if n == uint8(Syn) {
		return Syn, nil
	} else {
		return Invalid, fmt.Errorf("Invalid package type: %d", n)
	}
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
	ty, _ := extType(uint8(e.ty))
	return ty
}

func (e Ext) Iter() BitStream {
	return NewBitStream(e.data)
}

type PacketHeader struct {
	typeVer      uint8 // type: u4, ver: u4
	extension    uint8
	connectionId uint16

	// Both timestamps are in microseconds
	timestamp     uint32
	timestampDiff uint32

	wndSize uint32
	seqNr   uint16
	ackNr   uint16
}

func PckHdrFromByteSlice(b []byte) (*PacketHeader, error) {
	if len(b) < int(HeaderSize) {
		return nil, fmt.Errorf("Packet length: %d is less than %d", len(b), HeaderSize)
	}

	// check version. version info is in the lower nibble
	if b[0]&0x0f != 1 {
		return nil, fmt.Errorf("Unsupported version")
	}
	// packet type is specified by the higher nibble
	_, err := packetType(b[0] >> 4)
	if err != nil {
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

func NewPacket() Packet {
	return Packet{
		data: PacketHeader{}.asBytes(),
	}
}

// NewPacketWithPayloadcreates a new data packet, with appropriate header and provided payload
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

// comeback
func (p Packet) getExts() ExtIter {
	return ExtIter{}
}

type ExtIter struct {
}

func (p Packet) payload() []byte {
	return nil
}
