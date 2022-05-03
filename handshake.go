package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/OLUWAMUYIWA/odor/formats"
)

const PROTOCOL = "BitTorrent protocol"

// handshake: <pstrlen><pstr><reserved><info_hash><peer_id>

// pstrlen: string length of <pstr>, as a single raw byte
// pstr: string identifier of the protocol
// reserved: eight (8) reserved bytes. All current implementations use all zeroes.
// peer_id: 20-byte string used as a unique ID for the client.

type HandShake struct {
	infoHash formats.Sha1
	peerId   [20]byte
}

const pstr = "BitTorrent protocol"
const Null byte = 0

func NewHandShake(infoHash, peerId [20]byte) *HandShake {
	h := &HandShake{}
	h.infoHash = infoHash
	h.peerId = peerId

	return h
}

// Marshall marshalls an handshake object into a reader that can be read from
func (h *HandShake) Marshall() io.Reader {
	b := &bytes.Buffer{}
	b.Grow(49) // the spec says It is (49+len(pstr)) bytes long. 
	// write pstr len
	b.WriteByte(byte(len(PROTOCOL)))
	// write pstr
	b.WriteString(PROTOCOL)
	// write 8 mnull bytes. reserved
	b.Write( bytes.Repeat([]byte{Null}, 8))

	b.Write(h.infoHash[:])
	b.Write(h.peerId[:])
	return b
}

// ParseHandShake parses an handshake from a stream of bytes
func ParseHandShake(r io.Reader) (*HandShake, error) {
	//pstrLen
	var h *HandShake
	var pstrLen uint8
	if err := binary.Read(r, binary.BigEndian, pstrLen); err != nil {
		return nil, err
	}
	all, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(all) != 48+int(pstrLen) {
		return nil, errors.New("Handshake message flawed")
	}
	pstr := string(all[:pstrLen])
	if pstr != PROTOCOL {
		return nil, fmt.Errorf("We only support: %s", PROTOCOL)
	}
	//then the reserved 8 bytes
	h.infoHash = *((*[20]byte)(all[pstrLen+8 : 28+pstrLen]))
	h.peerId = *((*[20]byte)(all[28+pstrLen : 48+pstrLen]))
	return h, nil
}


