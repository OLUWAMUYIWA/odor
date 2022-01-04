package main

func nibble(p *Parser) (byte, error) {
	return p.ReadByte()
}

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

