package main

import (
	"errors"
	"fmt"
	"io"
)

type Info struct {
	name     []byte
	pieceLen int
	pieces   [][20]byte //muiltiple of twenty. SHAs of the piece at the corresponding index
	len      int        //length of the file in bytes
	path     []byte     //name of the file if it is a single file. name of the directory if it is a directory
}

type Torrent struct {
	announce []byte
	info     Info
}

func NewTorrent() *Torrent {
	return &Torrent{}
}

func (t *Torrent) Decode(r io.ReadCloser) error {
	parser, err := InitParser(r)
	if err != nil {
		return fmt.Errorf("Could not decode torrent file:  %w", err)
	}
	for {
		b, err := parser.ReadByte()
		if err != nil {
			if errors.Is(err, FileEndErr) { //nothing more to read. file has ended
				//comeback

			}
		}
		switch {
		case IsInt(b):
		case b == ':':
		case b == 'i':
		case b == 'd':
		case b == 'e':
		case b == 'l':
		default:
		}
	}
	defer r.Close()
	return nil
}

func (t *Torrent) Encode() ([]byte, error) {
	return nil, nil
}

func IsInt(b byte) bool {
	switch b {
	case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
		return true
	default:
		return false
	}
}
