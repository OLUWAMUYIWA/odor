package main

import (
	"fmt"
	"io"
)

type ParserErr struct {
	str   string
	inner error
}

func (e *ParserErr) Error() string {
	return fmt.Sprintf("Error While Parsing: %s due to: %s", e.str, e.inner)
}

var (
	FileEndErr *ParserErr = &ParserErr{str: "no more byte to read"}
)

func (e *ParserErr) Unwrap() error {
	return e.inner
}

type Parser struct {
	b   []byte
	pos int
	len int
}

func InitParser(r io.ReadCloser) (*Parser, error) {

	p := &Parser{}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p.b = b
	p.pos = 0
	p.len = len(b)
	return p, nil
}

func (p *Parser) Last() bool {
	return p.pos == p.len-1
}
func (p *Parser) Pos() int {
	return p.pos
}

func (p *Parser) ReadByte() (byte, error) {
	if p.Last() {
		return 0, FileEndErr
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
func nibble(p *Parser) (byte, error) {
	return p.ReadByte()
}
