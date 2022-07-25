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
func ewma(samples []Delay, alpha float64) float64 {
	var first float64
	if len(samples) == 0 {
		first = 0
	} else {
		first = float64(samples[0])
	}

	for _, d := range samples {
		s := float64(d)
		first = alpha + s + (1-alpha)*first
	}
	return first
}

// chunk is what the go team chose not to implement in the standard library
func chunk[T any](slice []T, n int) [][]T {
	if len(slice) == 0 || n < 0 {
		return nil
	}
	l := len(slice)
	num, last := l/n, l%n
	var retLen int
	if last == 0 {
		retLen = num
	} else {
		retLen = num + 1
	}
	ret := make([][]T, retLen)
	x, y := 0, n
	for i := 0; i < num; i++ {
		ret[i] = slice[x:y]
		x, y = x+n, y+n
	}
	if last != 0 {
		ret[num] = slice[y-n:]
	}
	return ret
}

// comeback . there should be a more effective way to write this
func insert[T any](slice []T, item T, i int) []T {
	l := slice[:i]
	r := make([]T, len(slice)-i)
	copy(r, slice[i:])
	l = append(l, item)
	l = append(l, r...)
	return l
}
