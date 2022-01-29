package parsec_test

import (
	"testing"
	"github.com/OLUWAMUYIWA/odor/parsec"
)


type ParsecErr struct {
	context string
	inner error
}



type TestInput struct {
	in []rune
}

func (i *TestInput) Car() rune {
	return (*i).in[0]
}

func (i *TestInput) Cdr() parsec.ParserInput {
	return &TestInput {
		in: (*i).in[1:],
	}
}

func (i *TestInput) Empty() bool {
	return len((*i).in) == 0
}


var  (
	Unmatched *ParsecErr = &ParsecErr{context: "Parser Unmatched"}
	Incomplete *ParsecErr = &ParsecErr{context: "There isn't enough data left fot this parser"}
)

func TestIsA(t *testing.T) {
	in := &TestInput {
		in: []rune{'a', 'b', 'c'},
	}
	actual := parsec.IsA('a')(in)
	if err,didErr := actual.Errored(); didErr {
		t.Errorf("Errored: %s", err)
	}
}