package utp

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

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

func randSeqID() (uint16, uint16) {
	rand.Seed(time.Now().Unix())
	id := uint16(rand.Int31()) // i just have to do this. math/rand cannot cannot generate 16bit integers
	if id == math.MaxUint16 {
		return id - 1, id
	} else {
		return id, id + 1
	}
}

// converts a big-endian integer to little-endian, or from little-endian to big-endian
func invEndUint32(i uint32) uint32 {
	return (i >> 24) | ((i >> 8) & 0x0000ff00) | ((i << 8) & 0x00ff0000) | ((i << 24) & 0xff000000)
}

func invEndUint16(i uint16) uint16 {
	return (i >> 8) | ((i << 8) & 0xff00)
}

func absDiff(a, b TimeStamp) Delay {
	if a > b {
		return Delay(a - b)
	} else {
		return Delay(b - a)
	}
}
