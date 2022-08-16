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
		t.Errorf("Wrong result\n")
	}
	rem := &BencInput{
		R: bufio.NewReader(bytes.NewBuffer([]byte{'v', 's'})),
	}
	actualRem, ok := res.Rem.(*BencInput)
	if !ok {
		t.Errorf("Not the type we expected")
	}
	if !reflect.DeepEqual(actualRem, rem) {
		t.Errorf("Type: expeced: %s\n", reflect.TypeOf(rem))
		t.Errorf("Type gotten: %s\n", reflect.TypeOf(actualRem))
		t.Errorf("Rem incorrect: should be: %s, but is: %s", rem, actualRem)
		for i :=0; i < 2; i++ {
			t.Errorf("Expect: %d. Got: %d\n", actualRem.Car(), rem.Car())
			rem = rem.Cdr().(*BencInput)
			actualRem = actualRem.Cdr().(*BencInput)
		}
		t.Errorf("got here\n")
	}
}

func TestBencStr(t *testing.T) {

}

func TestBencList(t *testing.T) {

}

func TestBenDict(t *testing.T) {

}
