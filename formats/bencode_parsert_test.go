package formats

import (
	"bufio"
	"bytes"
	"testing"
)

func TestBencInt(t *testing.T) {
	var b *BencInput = &BencInput{
		r: bufio.NewReader(bytes.NewBuffer([]byte{'i', '2', '2', '4', 'e', 'v', 's'})),
	}
	getInt := BencInt()
	num := 224

	res := getInt(b)
	if num != res.Result.(int) {
		t.Errorf("Error: %s", res.Err)
		t.Errorf("Expcted: %d. Got: %d", num, res.Result.(int))
	}
}

func TestBencStr(t *testing.T) {

}

func TestBencList(t *testing.T) {

}

func TestBenDict(t *testing.T) {

}
