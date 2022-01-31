package parsec

import (
	"reflect"
	"strings"
	"testing"
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