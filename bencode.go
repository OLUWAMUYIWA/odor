package main

import (
	"bufio"
	"bytes"
	"unicode/utf8"
)

type Bencode struct {
}

type Encoder struct {
	data []byte
}

type Decoder struct {
	data []byte
}



func (d *Decoder) Decode() {
	b := bufio.NewScanner(bytes.NewBuffer((*d).data))
	b.Split(bufio.ScanRunes)
	for b.Scan() {
		r, _ :=  utf8.DecodeRune(b.Bytes())
		
	}
}
