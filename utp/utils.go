package utp

import (
)

type BitStream struct {
	data []byte
	next, end int
}

func NewBitStream(b []byte) BitStream {
	return BitStream{
		data: b,
		next: 0,
		end: (len(b) * 8) -1,  // comeback last bit will be the
	}
}

func (b BitStream) CountOnes() int {
	return 1
}

func isBitSet(b byte, i int) bool {
	return (b & (1 << i)) == 1
}