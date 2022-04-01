package main

import (
	"net"
)

type Hash [20]byte

type Client struct {
	conn     net.Conn
	infoHash Hash
	peerId   Hash
	choked   bool
}
