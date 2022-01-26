package parsec_test

import (
	"testing"
	"github.com/OLUWAMUYIWA/odor/parsec"
)


type ParsecErr struct {
	context string
	inner error
}

type ParserInput interface {
	Car() rune //when it is called, it returns the current rune without advancing the index
	Cdr() ParserInput //returns the remainder of the input after the first one has been removed
	Empty() bool
}

type Input struct {
	in []rune
}

func (i *Input) Car() rune {
	return (*i).in[0]
}

func (i *Input) Cdr() *Input {
	return &Input {
		in: (*i).in[1:],
	}
}

func (i *Input) Empty() bool {
	return len((*i).in) == 0
}


var  (
	Unmatched *ParsecErr = &ParsecErr{context: "Parser Unmatched"}
	Incomplete *ParsecErr = &ParsecErr{context: "There isn't enough data left fot this parser"}
)

func TestIsA(t *testing.T) {
	actual := parsec.IsA('a')(&Input {
		in: []rune{'a', 'b', 'c'},
	})
}