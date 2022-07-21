package utp

import "fmt"

type BitStream struct {
	data      []byte
	next, end int
}

func NewBitStream(b []byte) BitStream {
	return BitStream{
		data: b,
		next: 0,
		end:  (len(b) * 8),
	}
}

// comeback
func (bs BitStream) CountOnes() int {
	n := 0
	for _, b := range bs.data {
		for i := 0; i < 8; i++ {
			if isBitSet(b, i) {
				n += 1
			}
		}
	}
	return n
}

func isBitSet(b byte, i int) bool {
	return (b & (1 << i)) == 1
}

func (b *BitStream) Next() (bool, error) {
	if b.end != b.next {
		byteIndex, bitIndex := b.next/8, byte(b.next%8)
		return (b.data[byteIndex] >> bitIndex & 0x01) == 1, nil
	}
	return false, fmt.Errorf("Ended, anything beyond this is out of range")
}
