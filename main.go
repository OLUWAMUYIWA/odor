package main

import (
	// "os"
	"bufio"
	"bytes"

	// "container/list"
	// "errors"
	"fmt"

	// "reflect"
	"strings"

	"github.com/OLUWAMUYIWA/odor/formats"
	"github.com/OLUWAMUYIWA/odor/parsec"
)

type TestInput struct {
	in []rune
}

func (i *TestInput) Car() rune {
	return (*i).in[0]
}

func (i *TestInput) Cdr() parsec.ParserInput {
	return &TestInput{
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
func main() {
	b := &formats.BencInput{R: bufio.NewReader(bytes.NewBuffer([]byte{'i', '2', '2', '4', 'e', 'v', 's'}))}
	fmt.Println("here")
	for !b.Empty() {
		fmt.Println("entered")
		fmt.Println("%s", b.Car())
		b = b.Cdr().(*formats.BencInput)
	}
	// g := parsec.Guarded('a', 'z')
	// in := &TestInput{in: []rune{'a', 'f', 'c'}}
	// res := g(in)
	// if err, did := res.Errored(); !did { // redundant
	// 	if !errors.Is(parsec.UnmatchedErr(), err) {
	// 		fmt.Printf("error should be unmatched but is %s", err.Error())
	// 	}
	// }

	// if res.Result != nil {
	// 	fmt.Printf("result should be nil")
	// }

	// if !reflect.DeepEqual(res.Rem.(*TestInput), &TestInput{[]rune{'a', 'f', 'c'}}) {
	// 	fmt.Printf("stream should not have been consumed")

	// }

	// in2 := &TestInput{in: []rune{'a', 'f', 'c'}}
	// t := parsec.TakeTill(func(r rune) bool { return r == 'c' })
	// res = t(in2)
	// fmt.Println(res.Result)
	// l, ok := res.Result.(*list.List)
	// if !ok {
	// 	fmt.Println("not a list")
	// }
	// for e := l.Front(); e != nil; e = e.Next() {
	// 	fmt.Println(e.Value.(rune))
	// }

	// driver := newDriver()
	// if err := driver.Drive(); err != nil {
	// 	os.Exit(1)
	// }
}
