package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/OLUWAMUYIWA/odor/formats"
)


type Client struct {
	conn     net.Conn
	infoHash formats.Sha1
	peerId   formats.Sha1
	choked   bool
}

const udpTimeout = time.Second * 5


// https://github.com/naim94a/udpt/wiki/The-BitTorrent-UDP-tracker-protocol

func Connect(ctx context.Context,  m formats.MetaInfo) (uint64, error) {
	rand.Seed(time.Now().Unix())
	txId := rand.Int31()

	raddr, err  := net.ResolveUDPAddr("udp4", m.Announce)

	if err != nil {
		return 0, err
	}
	c, err := net.DialUDP("udp4", nil, raddr)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	var b *bytes.Buffer
	initConnId := uint64(0x41727101980) // initial connection id
	initAction := uint32(0) // action number for connection request
	
	//set up connection request
	binary.Write(b, binary.BigEndian, &initConnId)
	binary.Write(b, binary.BigEndian, &initAction)
	binary.Write(b, binary.BigEndian, &txId)

	buff := make([]byte, 16)
	done := make(chan error, 1)

	go func() {
		_, err = io.Copy(c, b)
		if err != nil {
			done <- err
			return
		}

		err = c.SetReadDeadline(time.Now().Add(udpTimeout))
		if err != nil {
			done <- err
			return
		}

		_, _, err = c.ReadFromUDP(buff)
		if err != nil {
			done <- err
			return
		}
		done <- nil
	}()

	select {
	case <- ctx.Done():
		err = ctx.Err()
	case err = <-done:
	}

	respAction := binary.BigEndian.Uint32(buff[:4])
	if respAction != 0 {
		return 0, fmt.Errorf("Action should be zero")
	}
	respTx := binary.BigEndian.Uint32(buff[4:8])
	if int32(respTx) != txId {
		return 0, fmt.Errorf("Should be same like request's transaction id.")
	}
	connId := binary.BigEndian.Uint64(buff[8:])
	return connId, nil

}

func connReq() {

}

func Announce() {

}