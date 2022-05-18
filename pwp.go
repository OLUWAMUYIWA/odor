package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/OLUWAMUYIWA/odor/formats"
)

type Client struct {
	conn net.Conn
	infoHash formats.Sha1
	addr PeerAddr
	peerID [20]byte
	isChocked bool
	b formats.Bitfield
}


func Connect(ctx context.Context, addr PeerAddr) (*Client, error) {
	cl := &Client{}
	a := net.JoinHostPort(addr.ipv4.String(), strconv.Itoa(int(addr.port)))
	conn, err := net.DialTimeout("tcp", a, time.Second * 5)
	if err != nil {
		return nil, err
	}
	cl.conn = conn
	return cl, nil
}



func (c *Client) Shake(h *HandShake, infoHash formats.Sha1) (*HandShake, error) {
	// write the handshae to the connection
	if _, err := io.Copy(c.conn, h.Marshall()); err != nil {
		return nil, err
	}
	// read and parse the handshake response from the connection
	hRes, err := ParseHandShake(c.conn)
	if err != nil {
		return nil, err
	} 

	if bytes.Compare(infoHash[:], hRes.infoHash[:]) != 0 {
		return nil, fmt.Errorf("nvalid infoHash otten. expected: % x. Got % x", infoHash, hRes.infoHash)
	}

	return hRes, nil
}



func handleMsg(c net.Conn) {

}