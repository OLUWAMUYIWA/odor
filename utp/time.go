package utp

import (
	"fmt"
	"time"
)

type TimeStamp uint32

func nowMicroSecs() TimeStamp {
	tNano := time.Since(time.Unix(0, 0))
	tMicro := tNano.Microseconds()
	// comeback: we lose the higher four bytes in this conversion
	return TimeStamp(tMicro)
}

func NewTimeStamp() TimeStamp {
	return 0
}

func TimeStampFromInt(i int) TimeStamp {
	return TimeStamp(i)
}

func (d TimeStamp) Int() int {
	return int(d)
}

// Maximum age of base delay sample (60 seconds)
type Delay int64

func DelaFromU32(v uint32) Delay {
	return Delay(v)
}

func (d Delay) AsU32() uint32 {
	return uint32(d)
}

func (d Delay) AsI64() int64 {
	return int64(d)
}

func (d Delay) AsU64() uint64 {
	return uint64(d)
}

func NewDelay() Delay {
	return 0
}

func (d Delay) String() string {
	return fmt.Sprintf("Delay: %d", d)
}
