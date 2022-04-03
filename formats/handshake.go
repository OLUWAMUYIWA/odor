package formats

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type Sha1 [20]byte

// handshake: <pstrlen><pstr><reserved><info_hash><peer_id>

// pstrlen: string length of <pstr>, as a single raw byte
// pstr: string identifier of the protocol
// reserved: eight (8) reserved bytes. All current implementations use all zeroes.
// peer_id: 20-byte string used as a unique ID for the client.

type HandShake struct {
	pstr     string
	infoHash Sha1
	peerId   Sha1
}

const pstr = "BitTorrent protocol"

func New(infoHash, peerId Sha1) *HandShake {
	h := &HandShake{}
	h.pstr = pstr
	h.infoHash = infoHash
	h.peerId = peerId
	return h
}

func (h *HandShake) Marshall() io.Writer {
	b := &bytes.Buffer{}
	b.WriteByte(byte(len(h.pstr)))
	b.WriteString(h.pstr)
	b.Write(make([]byte, 8))
	b.Write(h.infoHash[:])
	b.Write(h.peerId[:])
	return b
}

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
	h.pstr = string(all[:pstrLen])
	//then the reserved 8 bytes
	h.infoHash = *((*[20]byte)(all[pstrLen+8 : 28+pstrLen]))
	h.peerId = *((*[20]byte)(all[28+pstrLen : 48+pstrLen]))
	return h, nil
}
