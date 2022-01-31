package parsec

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode"
)




type TestInput struct {
	in []rune
}

func (i *TestInput) Car() rune {
	return (*i).in[0]
}

func (i *TestInput) Cdr() ParserInput {
	return &TestInput {
		in: (*i).in[1:],
	}
}

func (i *TestInput) Empty() bool {
	return len((*i).in) == 0
}


func (i *TestInput) String() string {
	var s strings.Builder

	for _, r := range i.in {
		s.WriteRune(r)
	}

	return s.String()
}

func TestIsA(t *testing.T) {
	var inTable []*TestInput = []*TestInput{
		{[]rune {'a', 'b', 'c'}},
		{[]rune {'d', 'e', 'f'}},

	}

	expected := []rune{'a', 'd'}

	for i, r := range expected {
		res := IsA(r)(inTable[i])
		resR := res.Result.(rune)
		if resR != r {
			t.Errorf("IsA isn't popping the right rune \n")
		}
		inTable[i] = res.rem.(*TestInput)
	}


	if !reflect.DeepEqual(*inTable[0], TestInput{[]rune{'b', 'c'}}) {
		t.Errorf("Remander is not correct: %s instead of: %s \n", inTable[0], &TestInput{[]rune{'b', 'c'}})
	}

	if !reflect.DeepEqual(*inTable[1], TestInput{[]rune{'e', 'f'}}) {
		t.Errorf("Remander is not correct: %s instead of %s \n", inTable[1], &TestInput{[]rune{'e', 'f'}})
	}

}


func TestIsNot(t *testing.T) {
	var inTable []*TestInput = []*TestInput{
		{[]rune {'a', 'b', 'c'}},
		{[]rune {'d', 'e', 'f'}},

	}

	runes := []rune{'b', 'e'}

	for i, r := range runes {
		res := IsNot(r)(inTable[i])
		resR := res.Result.(rune)
		if res.err != nil {
			t.Errorf("Errored: %s", res.err)
		}
		if resR == r {
			t.Errorf("IsNot matches the said rune: %v instead of not doing so %v\n", resR, r)
		}
	}
}




func TestCharUTF8(t *testing.T) {
	//these guys ar invalid utf-8s, they should fail
	nonUTF8s := &TestInput{
		in: []rune("pythön!"),
	}

	for !nonUTF8s.Empty(){
		res := CharUTF8()(nonUTF8s)
		if e, did := res.Errored(); did {
			if !errors.Is(e, UnmatchedErr()) {
				t.Errorf("Sould be unmatched error")
			}
		}
		nonUTF8s = res.rem.(*TestInput)

	}

	nonUTF8s = &TestInput{
		in: []rune{0xe228a1, 0xe28228},
	}

	res := CharUTF8()(nonUTF8s)
	if res.Result != nil {
		t.Errorf("should be nill, but is: %v", res.Result)
	}

	if !reflect.DeepEqual(*nonUTF8s, *(res.rem.(*TestInput))) {
		t.Errorf("Should both be equal")
	}
}


func TestOneOf(t *testing.T) {
	in := TestInput {
		in: []rune {'d', 'e', 'f'},
	}	

	any := []rune{'d', 'a', 'b', 'c'}

	res := OneOf(any)(&in)

	if e, didErr := res.Errored(); didErr {
		t.Errorf("Errored when it shouldn't: %s", e)
	}

	if reflect.DeepEqual(*(res.rem.(*TestInput)), in) {
		t.Errorf("should not be equal beacuse it got reduced")
	}

	if !reflect.DeepEqual(TestInput{in: []rune{'e', 'f'}}, *(res.rem.(*TestInput))) {
		t.Errorf("should  be equal beacuse it got reduced")
	}
}


func TestDigit(t *testing.T) {
	in := &TestInput {
		in: []rune{'1', '2', '3', '4', '5', 'g'},
	}
	dig := Digit()
	expected := []int {1,2,3,4,5}

	for _, exp := range expected {
		res := dig(in)
		if res.Result.(int) != exp {
			t.Errorf("should be equal to the integers in expected. Should be: %d, but is %d", exp, res.Result.(int))
		}
		in = res.rem.(*TestInput)
	}
}

func TestIsEmpty(t *testing.T) {
	in := &TestInput {
		in: []rune{},
	}
	
	res := IsA('a')(in)

	if _, didErr := res.Errored(); !didErr {
		t.Errorf("Should have Errored but didn't")
	}

	if err, _ := res.Errored(); !errors.Is(err, IncompleteErr()) {
		t.Errorf("Error should have been %s, but is: %s", IncompleteErr(), err)
	}

}

func TestLetter(t *testing.T)  {
	in := &TestInput {
		in: []rune{'a', 'b', 'c'},
	}

	let := Letter()

	res := let(in)
	if r := res.Result.(rune); !unicode.IsLetter(r) {
		t.Errorf("Wrong!")
	}
}