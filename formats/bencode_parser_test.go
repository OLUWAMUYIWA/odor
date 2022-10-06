package formats

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
)

// note: can't believe i struggled with this test. `reflect.DeepEqual` just kept testing negative for actual and expected
// the reason, it turns out is that the two buffers, althoughhaving same contents had different lengths

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
	for i := 0; i < 2; i++ {
		a := actualRem.Car()
		b := rem.Car()
		if a != b {
			t.Errorf("Should be eqal. Actual: %b, expected: %b", a, b)
		}
		actualRem = actualRem.Cdr().(*BencInput)
		rem = rem.Cdr().(*BencInput)
	}
}
func TestBencStr(t *testing.T) {
	var b *BencInput = &BencInput{
		R: bufio.NewReader(bytes.NewBuffer([]byte{'4', ':', 's', 'p', 'a', 'm', '5'})),
	}
	getStr := BencStr()
	// expected := "spam"
	res := getStr(b)
	t.Errorf("Result: %v", res.Result)
	// if res.Result.(string) != expected {
	// 	t.Errorf("Expected: %s, got %s", expected, res.Result.(string))
	// }

}

func TestBencList(t *testing.T) {

}

func TestBenDict(t *testing.T) {

}
