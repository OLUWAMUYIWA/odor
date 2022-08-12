package formats

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
)

func TestBencInt(t *testing.T) {
	var b *BencInput = &BencInput{
		R: bufio.NewReader(bytes.NewBuffer([]byte{'i', '2', '2', '4', 'e', 'v', 's'})),
	}
	getInt := BencInt()
	num := 224

	res := getInt(b)
	nres, ok := res.Result.(int)
	if !ok {
		t.Errorf("Type: %s\n", reflect.TypeOf(res.Result))

	}
	if num != nres {
		t.Errorf("Error: %s", res.Err)
		t.Errorf("Expcted: %d. Got: %d", num, res.Result.(int))
	}
	rem := &BencInput{
		R: bufio.NewReader(bytes.NewBuffer([]byte{'v', 's'})),
	}
	actualRem := res.Rem.(*BencInput)
	if !reflect.DeepEqual(actualRem, rem) {
		t.Errorf("Type: expeced: %s\n", reflect.TypeOf(rem))
		t.Errorf("Type gotten: %s\n", reflect.TypeOf(actualRem))
		t.Errorf("Rem incorrect: should be: %s, but is: %s", rem, actualRem)
		for !actualRem.Empty() {
			t.Errorf("Expect: %s. Got:%s\n", b.Car())
			b = b.Cdr().(*BencInput)
		}
	}
}

func TestBencStr(t *testing.T) {

}

func TestBencList(t *testing.T) {

}

func TestBenDict(t *testing.T) {

}
