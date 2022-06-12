package main

import (
	"context"
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
	ChkdIntd     = Chkd | Intd
	ChkdUnIntd   = Chkd | UnIntd
	UnChkdIntd   = UnChkd | Intd
	UnchkdUnIntd = UnChkd | UnIntd
)

// PeerConn represents a connection between our client and another peer
type PeerConn struct {
	conn  net.Conn
	addr  PeerAddr
	state ConnState
	b     formats.Bitfield
	haves []int // if the peer does not use bitfield it must be using haves
}

// NewConn creates a tcp connection with a new per
func NewConn(ctx context.Context, addr PeerAddr) (*PeerConn, error) {
	cl := &PeerConn{}
	a := net.JoinHostPort(addr.ipv4.String(), strconv.Itoa(int(addr.port)))
	conn, err := net.DialTimeout("tcp", a, time.Second*5)
	if err != nil {
		return nil, err
	}
	cl.conn = conn
	return cl, nil
}

func (c *PeerConn) Shake(h *HandShake) error {
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
	msg, err := formats.ParseMessage(c.conn)
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

func (c *PeerConn) reqPiece(q Queue, p PiecesState) {
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

func handleMsg(c net.Conn, msg formats.Msg) {
	switch msg.ID {
	case formats.Choke:
		{
			c.Close()
		}
	case formats.Unchoke:
		{

		}
	case formats.Have:
		{

		}
	case formats.BitField:
		{

		}
	case formats.Piece:
		{

		}
	}
}
