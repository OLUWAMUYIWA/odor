package utp

import (
	"math"
	"math/rand"
	"time"
)

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

type number interface {
	int64 | int32 | int16 | int8 | int | uint | uint16 | uint8 | uint32 | uint64
}

func max[K number](x, y K) K {
	if x < y {
		return y
	}
	return x
}

func min[K number](x, y K) K {
	if x > y {
		return y
	}
	return x
}

func abs[K number](x K) K {
	if x < 0 {
		return 0 - x
	}
	return x
}

// ewma calculates the exponential weighted moving average for a vector of numbers, with a smoothing
// factor `alpha` between 0 and 1. A higher `alpha` discounts older observations faster.
func ewma() {

}
