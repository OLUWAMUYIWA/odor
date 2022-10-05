package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/OLUWAMUYIWA/odor/formats"
)

type ConnState uint8

const (
	Chkd ConnState = iota
	UnChkd
	Intd
	UnIntd
	ChkdIntd     = Chkd & Intd
	ChkdUnIntd   = Chkd & UnIntd
	UnChkdIntd   = UnChkd & Intd
	UnchkdUnIntd = UnChkd & UnIntd
)

// PeerConn represents a connection between our client and another peer
type PeerConn struct {
	conn  net.Conn
	addr  PeerAddr
	state struct {
		connState   ConnState
		index       int
		numReqsSent int
		buf         []byte
	}
	b     formats.Bitfield
	haves []int // if the peer does not use bitfield it must be using haves
}

// NewConn creates a tcp connection with a new peer
func NewConn(ctx context.Context, addr PeerAddr) (*PeerConn, error) {
	cl := &PeerConn{}
	a := net.JoinHostPort(addr.ipv4.String(), strconv.Itoa(int(addr.port)))
	conn, err := net.DialTimeout("tcp", a, time.Second*5)
	if err != nil {
		return nil, err
	}
	cl.conn = conn
	cl.addr = addr
	return cl, nil
}

func (c *PeerConn) Shake(h *Shaker) error {
	// write the handshake to the connection
	if _, err := io.Copy(c.conn, h.Marshall()); err != nil {
		return err
	}
	// read and parse the handshake response from the connection
	hRes, err := ParseHandShake(c.conn)
	if err != nil {
		return err
	}

	if !verifyhandShake(h, hRes) {
		return fmt.Errorf("Invalid infoHash gotten. expected: % x. Got % x", h.infoHash, hRes.infoHash)
	}

	return nil
}

func (c *PeerConn) ReqBitFields() error {
	c.conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer c.conn.SetDeadline(time.Time{})
	msg, err := formats.ReadMessage(c.conn)
	if err != nil {
		return err
	}
	if msg.ID != formats.BitField {
		return fmt.Errorf("Expected bitfield, got: %s", *msg)
	}
	b := formats.Bitfield(msg.Payload)
	c.b = b
	return nil
}

func (c PeerConn) HasPiece(i int) bool {
	return c.b.Has(i)
}

func (c *PeerConn) ReqPiece(q Queue, p PiecesState) {
	if q.chocked {
		return
	}
	for q.len() != 0 {
		pBlock := q.deq()
		if p.needed(pBlock) {
			req := formats.NewRequest(pBlock)
			req.Marshall(c.conn)
			p.assertReqd(pBlock)
			break
		}
	}
}

func (c *PeerConn) DownloadPiece(ctx context.Context, pReq *PieceReq, errchan chan error) {
	c.conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.conn.SetDeadline(time.Time{})

	numReqsSent := 0
	begin := 0
	var blockLen int = formats.BLOCK_LEN
	// request for all blocks
	for c.state.numReqsSent < pReq.len && begin < pReq.len {
		// if client is both unchoked and interested
		if c.state.connState == UnChkdIntd {
			// the last block is not bound to be same length as the first n blocks
			if pReq.len-begin < formats.BLOCK_LEN {
				blockLen = pReq.len - begin
			}
			if err := c.RequestBlock(ctx, pReq.index, begin, blockLen); err != nil {
				errchan <- err
				return
			}
			begin += blockLen
			numReqsSent++
		}
		msg, err := formats.ReadMessage(c.conn)
		if err != nil {
			errchan <- err
		}
		err = c.handleMsg(msg)

	}

}

func (c *PeerConn) Unchoke() error {
	unchoke := formats.NewUnchoke()
	return unchoke.Marshall(c.conn)
}
func (c *PeerConn) Interested() error {
	unchoke := formats.NewIntd()
	return unchoke.Marshall(c.conn)
}
func (c *PeerConn) Uninterested() error {
	unchoke := formats.NewUnIntd()
	return unchoke.Marshall(c.conn)
}

func (c *PeerConn) RequestBlock(ctx context.Context, index int, begin int, length int) error {
	req := formats.NewRequest(formats.Ibl{Index: index, Begin: begin, Length: length})
	return req.Marshall(c.conn)
}

func (c *PeerConn) handleMsg(msg *formats.Msg) error {
	switch msg.ID {
	case formats.Choke:
		{
			c.state.connState = Chkd
			return c.conn.Close()
		}
	case formats.Unchoke:
		{
			c.state.connState = UnChkd
			return nil
		}
	case formats.Have:
		{
			if msg.ID != formats.Have {
				return fmt.Errorf("Error parsing Have message, incorrect ID")
			}
			if len(msg.Payload) != 4 {
				return fmt.Errorf("Payload length should be 4, but is: %d", len(msg.Payload))
			}
			i := binary.BigEndian.Uint32(msg.Payload)
			c.b.Set(int(i))
			return nil
		}
	case formats.Piece:
		{
			_, err := formats.ParsePieceMsg(msg)
			if err != nil {
				return err
			}

			return nil
		}
		// comeback
	default:
		{
			return nil
		}
	}

}
