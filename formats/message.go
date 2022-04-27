package formats

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	// hack: i made keep_alive id 10
	KepAlive
)

type Message struct {
	id     uint8
	buffer *bytes.Buffer
}

type HaveIndex uint32

type Bitfield []byte

func (b Bitfield) Set(i int) error {
	if i < 0 {
		return fmt.Errorf("Out of bounds")
	}
	pos := i / 8 // byte position
	off := i % 8 // offset in byte position
	if pos < 0 || pos >= len(b) {
		return fmt.Errorf("Out of bounds")
	}
	b[pos] = b[pos] | (1 << uint(7 - off))
	return nil
}

func (b Bitfield) Has(i int) bool {
	if i < 0 {
		return false
	}
	pos := i / 8
	off := i % 8
	if pos >= len(b) {
		return false
	}
	return b[pos]>>uint(7-off)&1 != 0
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
	case KepAlive: {

	}
	}
	return m
}

func (m *Message) Marshall(w io.Writer) error  {
	_, err := io.Copy(w, m.buffer)
	return err
}


func ParseMessage(r io.Reader) (*MsgId, []byte, error) {
	lBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lBuf); err != nil {
		return nil, nil, err
	}
	l := binary.BigEndian.Uint32(lBuf)


	if l == 0 {
		// comeback: hac: i made id for keep-alive to be 10
		var id MsgId = 10
		return &id, nil, nil
	}
	msg := make([]byte, l)
	if _, err := io.ReadFull(r, msg); err != nil {
		return nil, nil, err
	}

	id, payload  := MsgId(msg[0]), msg[1:]

	idRef := &id

	return idRef, payload, nil
}

// const connectionID uint64 = 0x41727101980

// func buildConnReq() (io.Reader, error) {
// 	var b bytes.Buffer
// 	connIdBytes := []byte{}
// 	binary.BigEndian.PutUint64(connIdBytes, connectionID)
// 	n, err := b.Write(connIdBytes)
// 	if err != nil || n != 8 {
// 		return nil, err
// 	}
// 	b.Write([]byte{0, 0, 0, 0})
// 	var random [4]byte
// 	_, _ = rand.Read(random[:])
// 	io.ReadFull(&b, random[:])
// 	return &b, nil
// }
