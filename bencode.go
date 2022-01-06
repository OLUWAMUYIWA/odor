package main

type Bencode struct {
}

type Encoder struct {
}

type Decoder struct {
}

func nibble(p *Parser) (byte, error) {
	return p.ReadByte()
}
