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

const PORT = 6881

type Client struct {
	infoHash formats.Sha1
	peerId   formats.Sha1
	announce string
	conn *net.UDPConn
}

const udpTimeout = time.Second * 5


// https://github.com/naim94a/udpt/wiki/The-BitTorrent-UDP-tracker-protocol

func (client *Client) Connect(ctx context.Context) (uint64, error) {
	rand.Seed(time.Now().Unix())
	txId := rand.Int31()

	raddr, err  := net.ResolveUDPAddr("udp4", client.announce)

	if err != nil {
		return 0, err
	}
	c, err := net.DialUDP("udp4", nil, raddr)
	client.conn = c
	if err != nil {
		return 0, err
	}
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


func (client *Client) Announce(ctx context.Context,  connId uint64, size uint64) (*AnnounceResp, error) {
	var b bytes.Buffer
	buf := make([]byte, 8)  //reusable buffer
	binary.BigEndian.PutUint64(buf, connId)
	b.Write(buf) //write connId from server
	binary.BigEndian.PutUint32(buf[0:4], uint32(1))
	b.Write(buf[:4]) // write action number forannounce
	txId := rand.Int31()
	binary.BigEndian.PutUint32(buf[:4], uint32(txId))
	b.Write(buf[:4]) // write new transaction ID
	b.Write(client.infoHash[:]) // write the info_hash of the torrent that is being announced
	b.Write(client.peerId[:]) // write the peer ID of the client announcing itself
	empty := make([]byte, 8, 8)
	b.Write(empty) // write bytes downloaded by client this session
	binary.BigEndian.PutUint64(buf, size)
	b.Write(buf) // write bytes left to complete the download
	b.Write(empty) // write bytes uploaded this session
	binary.BigEndian.PutUint32(buf[:4], uint32(0))
	b.Write(buf[:4]) // event. zero for none
	b.Write(empty) // write IP address, default set to 0  
	key := make([]byte, 4)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	b.Write(key)
	var minus1 int32 = -1
	binary.BigEndian.PutUint32(buf[:4], uint32(minus1)) // write num-want: -1 by default. number of clients to return
	b.Write(buf[:4])
	binary.BigEndian.PutUint16(buf[:2], uint16(PORT))
	b.Write(buf[:2])

	c := client.conn

	resp := make([]byte, 1024)	
	done := make(chan error, 1)
	var a *AnnounceResp
	go func () {
		_, err = io.Copy(c, &b)
		if err != nil {
			done <- err
			return
		}

		err = c.SetReadDeadline(time.Now().Add(udpTimeout))
		if err != nil {
			done <- err
			return
		}
		
		_, _, err = c.ReadFromUDP(resp)
		if err != nil {
			done <- err
			return
		}
	}()

	select{
	case <- ctx.Done():
		err = ctx.Err()
	case err = <-done :
	}


	a, err = ParseAnnounceResp(resp)
	if err != nil {
		return nil, err
	}

	return a, err
}

type AnnounceResp struct {
	txId uint32
	interval uint32
	leechers uint32
	seeders uint32
	ipv4 net.IP
	port uint16
}


// comeback
func ParseAnnounceResp(b []byte) (*AnnounceResp, error) {
	return nil, nil
}