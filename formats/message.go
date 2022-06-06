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
	Len     int
	ID      MsgId
	Payload []byte
}

// Stringer impl of Msg so i can print out the type
func (m Msg) String() string {
	switch m.ID {
	case Choke:
		return "Choke"
	case Unchoke:
		return "Unchoke"
	case Interested:
		return "Interested"
	case Uninterested:
		return "Uninterested"
	case Have:
		return ""
	case BitField:
		return "BitField"
	case Request:
		return "Request"
	case Piece:
		return "Piece"
	case Cancel:
		return "Cancel"
	case Port:
		return "Port"
	default:
		return "Unknown"
	}
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
	b[pos] = b[pos] | (1 << uint(7-off))
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
	Index, Begin, Length uint32
}

func (m *Msg) Marshall(w io.Writer) error {
	switch m.ID {
	case Choke | Unchoke | Interested | Uninterested: // <len=0001><id=x>
		{
			//length
			b := make([]byte, 5)
			binary.BigEndian.PutUint32(b[:4], uint32(1))
			b[4] = uint8(m.ID)
			_, err := w.Write(b)
			return err
		}
	case Have: // have: <len=0005><id=4><piece index>
		{
			b := make([]byte, 9)
			binary.BigEndian.PutUint32(b[:4], uint32(5))
			b[4] = uint8(m.ID)
			n := copy(b[4:], m.Payload)
			if n != 4 {
				return fmt.Errorf("`Have`s payload should be four bytes long")
			}
			if _, err := w.Write(b); err != nil {
				return err
			}
		}
	case BitField: // bitfield: <len=0001+X><id=5><bitfield>
		{
			l := len(m.Payload) + 1
			buf := make([]byte, l+4)
			binary.BigEndian.PutUint32(buf[:4], uint32(l))
			buf[4] = byte(m.ID)
			copy(buf[5:], m.Payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			}
		}
	case Request: // request: <len=0013><id=6><index><begin><length>
		{
			buf := make([]byte, 17)
			binary.BigEndian.PutUint32(buf[:4], 13)
			buf[4] = byte(m.ID)
			copy(buf[5:], m.Payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			}
		}
	case Piece: // piece: <len=0009+X><id=7><index><begin><block>
		{
			l := len(m.Payload) + 1
			buf := make([]byte, l+4)
			binary.BigEndian.PutUint32(buf[:4], uint32(l))
			buf[4] = byte(m.ID)
			copy(buf[5:], m.Payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			}

		}
	case Cancel: // <len=0013><id=8><index><begin><length>
		{
			buf := make([]byte, 17)
			binary.BigEndian.PutUint32(buf[:4], 13)
			buf[4] = byte(m.ID)
			copy(buf[5:], m.Payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			}
		}
	case Port: // <len=0003><id=9><listen-port>
		{
			buf := make([]byte, 7)
			binary.BigEndian.PutUint32(buf[:4], 3)
			buf[4] = byte(m.ID)
			copy(buf[5:], m.Payload)
			_, err := w.Write(buf)
			if err != nil {
				return err
			}
		}
	case KepAlive:
		{
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
		return &Msg{Len: l, ID: KepAlive, Payload: []byte{}}, nil
	}

	msg := make([]byte, l)
	if _, err := io.ReadFull(r, msg); err != nil {
		return nil, err
	}

	id := MsgId(msg[0])
	if l == 1 {
		return &Msg{Len: 1, ID: id, Payload: []byte{}}, nil
	}
	payload := msg[1:]

	m.ID = MsgId(id)
	m.Len = l
	m.Payload = payload
	return m, nil
}

func NewChoke() *Msg {
	m := &Msg{}
	m.ID = Choke
	m.Len = 1
	return m
}

func NewUnchoke() *Msg {
	m := &Msg{}
	m.ID = Unchoke
	m.Len = 1
	return m
}

func NewIntd() *Msg {
	m := &Msg{}
	m.ID = Interested
	m.Len = 1
	return m
}

func NewUnIntd() *Msg {
	m := &Msg{}
	m.ID = Uninterested
	m.Len = 1
	return m
}

func NewHave(pieceIndex uint32) *Msg {
	m := &Msg{}
	m.ID = Have
	m.Len = 5
	p := make([]byte, 4)
	binary.BigEndian.PutUint32(p, pieceIndex)
	return m
}

// Ibl Index-Begin-Length trio data structure
type Ibl struct {
	Index, Begin, Length int
}

func NewRequest(ibl Ibl) *Msg {
	m := &Msg{}
	m.ID = Request
	m.Len = 13
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(ibl.Index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(ibl.Begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(ibl.Length))
	m.Payload = payload
	return m
}

type PieceMsg struct {
	Index, Begin uint32
	Block        []byte
}

func NewPieceMMsg(p PieceMsg) *Msg {
	m := &Msg{}
	m.ID = Request
	m.Len = 9 + len(p.Block)
	payload := make([]byte, 8+len(p.Block))
	binary.BigEndian.PutUint32(payload[0:4], p.Index)
	binary.BigEndian.PutUint32(payload[4:8], p.Begin)
	copy(payload, p.Block)
	m.Payload = payload
	return m
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
