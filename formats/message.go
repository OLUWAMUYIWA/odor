package formats

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type MsgId uint8

type Sha1 [20]byte

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

// Msg: All of the remaining messages in the protocol take the form of <length prefix><message ID><payload>
type Msg struct {
	len int
	id MsgId
	payload []byte
}

type MsgDec struct {
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

func (m *Msg) Marshall(w io.Writer) error {
	switch m.id {
	case Choke | Unchoke | Interested | Uninterested: // <len=0001><id=x>
		{
			//length
			b := make([]byte, 5)
			binary.BigEndian.PutUint32(b[:4], uint32(1))
			b[4] = uint8(m.id)
			_, err := w.Write(b)
			return err
		}
	case Have: // have: <len=0005><id=4><piece index>
		{
			b := make([]byte, 9)
			binary.BigEndian.PutUint32(b[:4], uint32(5))
			b[4] = uint8(m.id)
			n := copy(b[4:], m.payload)
			if n != 4 {
				return fmt.Errorf("`Have`s payload should be four bytes long")
			}
			if _, err := w.Write(b); err != nil {
				return err
			}
		}
	case BitField: // bitfield: <len=0001+X><id=5><bitfield>
		{
			l := len(m.payload) + 1
			buf := make([]byte, l+4)
			binary.BigEndian.PutUint32(buf[:4], uint32(l))
			buf[4] = byte(m.id)
			copy(buf[5:], m.payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			} 
		}
	case Request: // request: <len=0013><id=6><index><begin><length>
		{
			buf := make([]byte, 17)
			binary.BigEndian.PutUint32(buf[:4], 13)
			buf[4] = byte(m.id)
			copy(buf[5:], m.payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			} 
		}
	case Piece: // piece: <len=0009+X><id=7><index><begin><block>
		{
			l := len(m.payload) + 1
			buf := make([]byte, l + 4)
			binary.BigEndian.PutUint32(buf[:4], uint32(l))
			buf[4] = byte(m.id)
			copy(buf[5:], m.payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			} 

		}
	case Cancel: // <len=0013><id=8><index><begin><length>
		{
			buf := make([]byte, 17)
			binary.BigEndian.PutUint32(buf[:4], 13)
			buf[4] = byte(m.id)
			copy(buf[5:], m.payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			} 
		}
	case Port: // <len=0003><id=9><listen-port>
		{
			buf := make([]byte, 7)
			binary.BigEndian.PutUint32(buf[:4], 3)
			buf[4] = byte(m.id)
			copy(buf[5:], m.payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			} 
		}
	case KepAlive: {
		if _, err := w.Write(bytes.Repeat([]byte{0}, 4)); err != nil {
			return err
		}
	}
	}
	return nil
}


func ParseMessage(r io.Reader) (*Msg, error) {
	m := &Msg{}
	lBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lBuf); err != nil {
		return nil, err
	}
	l := int(binary.BigEndian.Uint32(lBuf))


	if l == 0 {
		// comeback: hack: i made id for keep-alive to be 10
		return &Msg{len: l, id: KepAlive, payload: []byte{}}, nil
	}

	msg := make([]byte, l)
	if _, err := io.ReadFull(r, msg); err != nil {
		return nil, err
	}

	id  := MsgId(msg[0]) 
	if l == 1 {
		return &Msg{len: 1, id: id, payload: []byte{}}, nil
	}
	payload := msg[1:]

	m.id = MsgId(id)
	m.len = l
	m.payload = payload
	return m, nil
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
