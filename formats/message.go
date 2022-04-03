package formats

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/rand"
)

type MsgId uint8

const (
	Choke MsgId = iota
	Unchoke
	Interested
	Uninterested
	Have
	BitField
	Request
	Piece
	Cancel
	Port
)

type Message struct {
	id     uint8
	buffer *bytes.Buffer
}

type HaveIndex uint32

type Bitfield []byte

type MessageCont interface {
	HaveIndex | Bitfield 
}

type Payload struct {
	index, begin, length uint32

}

func NewMessage(id MsgId, payload any) *Message {
	m := &Message{
		buffer: &bytes.Buffer{},
	}
	switch id {
	case Choke | Unchoke | Interested | Uninterested:
		{
			m.id = uint8(Choke)
			//length
			l := uint32(1)
			binary.Write(m.buffer, binary.BigEndian, &l)
			id := uint8(id)
			binary.Write(m.buffer, binary.BigEndian, &id)
			return m
		}
	case Have:
		{
			m.id = uint8(Have)
			l := uint32(5)
			binary.Write(m.buffer, binary.BigEndian, &l)
			id := uint8(Have)
			binary.Write(m.buffer, binary.BigEndian, &id)
			index, ok := payload.(uint32)
			if !ok {
				panic("should be called with a uint32")
			}
			binary.Write(m.buffer, binary.BigEndian, &index)
		}
	case BitField:
		{
			p,ok := payload.(Payload)
			if !ok {
				panic("Expects a valid Payload object")
			}
			//length
			binary.Write(m.buffer, binary.BigEndian, &(p.length))
			id := uint8(BitField)
			binary.Write(m.buffer, binary.BigEndian, &id)
			// comeback
		}
	case Request:
		{
			l := uint32(13)
			binary.Write(m.buffer, binary.BigEndian, &l)
			id := uint8(Request)
			binary.Write(m.buffer, binary.BigEndian, &id)
			p,ok := payload.(Payload)
			if !ok {
				panic("Expects a valid Payload object ")
			}
			binary.Write(m.buffer, binary.BigEndian, &(p.index))
			binary.Write(m.buffer, binary.BigEndian, &(p.begin))
			binary.Write(m.buffer, binary.BigEndian, &(p.length))

		}
	case Piece:
		{
			
		}
	case Cancel:
		{

		}
	case Port:
		{

		}
	}
	return m
}

func (m *Message) Marshall() io.Writer {
	var b bytes.Buffer

	return &b
}

const connectionID uint64 = 0x41727101980

func buildConnReq() (io.Reader, error) {
	var b bytes.Buffer
	connIdBytes := []byte{}
	binary.BigEndian.PutUint64(connIdBytes, connectionID)
	n, err := b.Write(connIdBytes)
	if err != nil || n != 8 {
		return nil, err
	}
	b.Write([]byte{0, 0, 0, 0})
	var random [4]byte
	_, _ = rand.Read(random[:])
	io.ReadFull(&b, random[:])
	return &b, nil
}

func parseConnResp(r io.Reader) {

}
