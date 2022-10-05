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

func (m MsgId) String() string {
	switch m {
	case Choke:
		return "0"
	case Unchoke:
		return "1"
	case Interested:
		return "2"
	case Uninterested:
		return "3"
	case Have:
		return "4"
	case BitField:
		return "5"
	case Request:
		return "6"
	case Piece:
		return "7"
	case Cancel:
		return "8"
	case Port:
		return "9"
	default:
		return "10"
	}
}

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
		return "Choke {Id: 0}"
	case Unchoke:
		return "Unchoke {Id: 1}"
	case Interested:
		return "Interested {Id: 2}"
	case Uninterested:
		return "Uninterested {Id: 3}"
	case Have:
		return "Have {Id: 4}"
	case BitField:
		return "BitField {Id: 5}"
	case Request:
		return "Request {Id: 6}"
	case Piece:
		return "Piece {Id: 7}"
	case Cancel:
		return "Cancel {Id: 8}"
	case Port:
		return "Port {Id: 9}"
	case KepAlive:
		return "KeepAlive"
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

// Marshall marshalls any constructed message into a writer. The type of message, specified by the `ID` determines how it is marshalled
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
			return nil
		}
	}
	return nil
}

// ReadMessage reads from an `io.Reader`, usually a client connection, and puts the bytes into the generic `Msg` struct.
// From `Msg` we can further parse into different message types
func ReadMessage(r io.Reader) (*Msg, error) {
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

// NewRequest: creates a request for a block (part of a piece). Its twin is the Piece message
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

// NewCancel: Cancel messages are generally only sent towards the end of a download, during what's called 'endgame mode'.
func NewCancel(ibl Ibl) *Msg {
	m := &Msg{}
	m.ID = Cancel
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

func ParsePieceMsg(msg *Msg) (PieceMsg, error) {
	if msg.ID != Piece {
		return PieceMsg{}, fmt.Errorf("Expected %s, got ID %d", Piece, msg.ID)
	}
	if len(msg.Payload) < 8 {
		return PieceMsg{}, fmt.Errorf("Message content too short for Piece Message")
	}
	// first 4 bytes of the payload is for the index
	index := binary.BigEndian.Uint32(msg.Payload[0:4])
	// next four bytes is for the `begin`
	begin := binary.BigEndian.Uint32(msg.Payload[4:8])
	// the bytes itself take the remaining
	buf := msg.Payload[8:]
	return PieceMsg{Index: (index), Begin: begin, Block: buf}, nil
}

// NewPieceMMsg creates a new marshallable `Msg` from
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
