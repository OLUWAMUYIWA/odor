package main

import (
	"fmt"
	"io"
)

type Parser struct {
	b   []byte
	pos int
	len int
}

func (p *Parser) Init(r io.ReadCloser) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	p.b = b
	p.pos = 0
	p.len = len(b)
	return nil
}

func (p *Parser) Last() bool {
	return p.pos == p.len-1
}
func (p *Parser) Pos() int {
	return p.pos
}

func (p *Parser) ReadByte() (byte, error) {
	if p.Last() {
		return 0, fmt.Errorf("nothing more to read")
	}
	ret := p.b[p.pos]
	p.pos++
	return ret, nil
}

//read the next byte without incrementing the position
func (p *Parser) Peep() (byte, error) {
	if p.Last() {
		return 0, fmt.Errorf("nothing more to read")
	}
	return p.b[p.pos], nil
}
