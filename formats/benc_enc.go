package formats

import (
	"io"
)


type BencEncoder struct {
	wtr io.Writer
}

func NewBencoder(wtr io.Writer) *BencEncoder {
	return &BencEncoder{
		wtr: wtr,
	}
}

func (b *BencEncoder) Encode(v any) error {
	return nil
}

