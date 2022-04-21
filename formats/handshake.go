package formats

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

const PROTOCOL = "BitTorrent protocol"

type Sha1 [20]byte

// handshake: <pstrlen><pstr><reserved><info_hash><peer_id>

// pstrlen: string length of <pstr>, as a single raw byte
// pstr: string identifier of the protocol
// reserved: eight (8) reserved bytes. All current implementations use all zeroes.
// peer_id: 20-byte string used as a unique ID for the client.

type HandShake struct {
	infoHash Sha1
	peerId   Sha1
}

const pstr = "BitTorrent protocol"

func NewHandShake(infoHash, peerId Sha1) *HandShake {
	h := &HandShake{}
	h.infoHash = infoHash
	h.peerId = peerId

	return h
}

// Marshall marshalls an handshake object into a reader that can be read from
func (h *HandShake) Marshall() io.Reader {
	b := &bytes.Buffer{}
	b.WriteByte(byte(len(PROTOCOL)))
	b.WriteString(PROTOCOL)
	b.Write(make([]byte, 8))
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


func (h *HandShake) Shake(conn net.Conn, infoHash Sha1) (*HandShake, error) {
	if _, err := io.Copy(conn, h.Marshall()); err != nil {
		return nil, err
	}

	hRes, err := ParseHandShake(conn)
	if err != nil {
		return nil, err
	} 

	if bytes.Compare(infoHash[:], hRes.infoHash[:]) != 0 {
		// comeback
		return nil, fmt.Errorf("nvalid infoHash otten. expected: % x. Got % x", infoHash, hRes.infoHash)
	}

	return hRes, nil
}