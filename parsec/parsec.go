//parsec is a mini parser combinator library
//It does more than i need it to do for bencode parsing, but I decided to make it bigger than necessary because i wanted
//to learn a little more about writing parser combinators and get more familiar with the functional style of programming in go

package parsec

import (
	"container/list"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

//Parsec is a basic parser function. It takes an imput and returns PResult as result.
type Parsec func (in ParserInput) PResult

//Predicate is a function that takes a rune and performs some computation, returning a true/false result
//this result is useful when the predicate is used a a function argument is a higher order function.
//a true result proves that the rune in question satisfies a particular condition
type Predicate func(r rune) bool

//ParserInput specifies two methods.
//The method `Car` returns the next rune in the stream.. an implememter only needs return the first item in its list when Car is called
//Cdr OTOH, while also not changing the internal state of the implementer, returns another copy of the implementer
//without the first part. It works like a `Lisp`
type ParserInput interface {
	Car() rune //when it is called, it returns the current rune without advancing the index
	Cdr() ParserInput //returns the remainder of the input after the first one has been removed
	Empty() bool
}

//PResult contains two fields. `result` contains the result of the parser. `rem` contains the remaining imput
//if the parser succeeds then `rem` is the remainder of the input after the `matched` runes have been moved out of it
//if the parser fails, the rem contains the input unchanged
type PResult struct {
	result interface{}
	rem    ParserInput
	err error
}

type ParsecErr struct {
	context string
	inner error
}

func (e *ParsecErr) Error() string {
	return fmt.Sprintf("Error: %s\n Reason: %s", e.context, e.inner)
}

func (e *ParsecErr) Unwrap() error {
	return e.inner
}


////////SIMPLE PARSERS
// IsA is the simplest parser, it checks if a rune matches the next rune in the input.
func IsA(r rune) Parsec {
	return func (in ParserInput) PResult {
		if !in.Empty() && r == in.Car() { 
			return PResult{r, in.Cdr()}
		}

		return PResult{
			nil, in,
		}
	}
}

// IsNot is the complete opposite of IsA.
func IsNot(r rune) Parsec {
	return func(in ParserInput) PResult {
		if !in.Empty() && r == in.Car() {
			return PResult{nil, in}
		}
		return PResult{r, in.Cdr()}
	}
}

//checks if this rune is a valid utf-8 character. thhis character could be any utf-8 symbol
func CharUTF8(c rune) Parsec {
	return func(in ParserInput) PResult {
		curr := in.Car()
		if !in.Empty() && utf8.ValidRune(curr) {
			return PResult{
				curr, in.Cdr(),
			}
		}
		return PResult{nil, in}
	}
}

func OneOf(any []rune) Parsec {
	return func(in ParserInput) PResult {
		for _, r := range any {
			curr := in.Car()
			if curr == r {
				return PResult{
					r,
					in.Cdr(),
				}
			}
		}
		return PResult{
			nil,
			in,
		}
	}
}

//digit takes only utf-8 encoded runes and ensures they are decimal digits (0-9)
func Digit() Parsec {
	return func(in ParserInput) PResult {
		curr := in.Car()
		//if curr is a unicode number
		if !in.Empty() && utf8.ValidRune(curr) && unicode.IsDigit(curr) {
			return PResult{
				curr, in.Cdr(),
			}
		}
		//else
		return PResult{nil, in}
	}
}

//letter takes only utf-8 encoded runes and ensures they are letters
func Letter(c rune) Parsec {
	return func(in ParserInput) PResult {
		curr := in.Car()
		if !in.Empty() && utf8.ValidRune(curr) && unicode.IsLetter(curr) {
			return PResult{
				curr,
				in.Cdr(),
			}
		}
		return PResult{
			nil,
			in,
		}
	}
}

/////REPETITIONS

//Take eats up `n` number of runes. if it doesnt get up to `n` number, it fails
func Take(n int) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{
				nil,
				in,
			}
		}
		rem := in //rem needs to have inout copied into it because we want to retain te full input, in case of a failure where we return the entire input
		res := list.New()
		//first one. we're sure this will yield a value bwcause the input isn't empty
		first := rem.Car()
		res.PushBack(first)
		for i := 0; i < n-1; i++ { //the upper limit is n-1 because we have already taken one (first)
			if rem.Empty() { //we exhausted the input before taking all we wanted
				return PResult{
					nil,
					in,
				}
			} else { //there's more, and we haven't reached our target number
				res.PushBack(in.Car())
				rem = rem.Cdr()
			}

		}

		if res.Len() < n { //doublecheck
			return PResult{
				nil,
				in,
			}
		}
		return PResult{
			res,
			rem,
		}
	}
}

//TakeTill eats runes until a Predicate is satisfied
func TakeTill(f Predicate) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{
				nil,
				in,
			}
		}
		rem := in
		res := list.New()
		curr := rem.Car()

		for !rem.Empty() {
			if f(curr) {
				if res.Len() == 0 { //empty
					return PResult{
						nil,
						in,
					}
				}
				return PResult{
					res,
					rem,
				}
			}
			res.PushBack(curr)
			rem = rem.Cdr()
			curr = rem.Car()
		}
		if res.Len() == 0 { //empty
			return PResult{
				nil,
				in,
			}
		}
		return PResult{
			res,
			rem,
		}
	}
}

//TakeWhile keeps eating runes while Pedicate returns true
func TakeWhile(f Predicate) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{
				nil,
				in,
			}
		}
		rem := in
		res := list.New()
		curr := rem.Car()
		for !rem.Empty() && f(curr) {
			res.PushBack(curr)
			rem = rem.Cdr()
			curr = rem.Car()
		}

		if res.Len() == 0 {
			return PResult{
				nil,
				in,
			}
		}
		return PResult{
			res,
			rem,
		}
	}
}

func Terminated(match, post string) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{
				nil,
				in,
			}
		}
		rem := in
		matchRunes, postRunes := []rune(match), []rune(post) //create rune slices from the strings
		for _, r := range matchRunes {
			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
				}
			}
			rem = rem.Cdr()
			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
				}
			}
		}
		for _, r := range postRunes {
			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
				}
			}
			rem = rem.Cdr()
			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
				}
			}
		}

		return PResult{
			match,
			rem,
		}
	}
}

func Preceded(match, pre string) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{
				nil,
				in,
			}
		}
		rem := in
		matchRunes, preRunes := []rune(match), []rune(pre) //create rune slices from the strings
		for _, r := range preRunes {
			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
				}
			}
			rem = rem.Cdr()
			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
				}
			}
		}
		for _, r := range matchRunes {
			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
				}
			}
			rem = rem.Cdr()
			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
				}
			}
		}

		return PResult{
			match,
			rem,
		}
	}
}

func Number() Parsec {
	return func(in ParserInput) PResult {
		var numStr strings.Builder
		numbers := []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
		digs := OneOf(numbers)
		rem := in
		for !rem.Empty() {
			res := digs(rem)
			if res.result != nil {
				if s, ok := res.result.(rune); ok {
					numStr.WriteRune(s)
					rem = rem.Cdr()
				}
			} else {
				break
			}
		}

		if numStr.String() == "" {
			return PResult{
				nil,
				in,
			}
		}
		ans, _ := strconv.Atoi(numStr.String()) // there can never be an error.
		return PResult{
			ans,
			in,
		}
	}
}

//Chars asks if a stream of input matches the characters in the rune slice provided
//if it doesn't te entire input is returned unchanged, but with a nil result
func Chars(chars []rune) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in}
		}
		rem := in //remainder is fist the entire input
		for _, c := range chars {
			res := CharUTF8(c)(rem)
			if res.result == nil { //parser failed to match
				return PResult{
					nil, rem,
				}
			}
			rem = res.rem
		}
		return PResult{chars, rem}
	}
}

//Str is a special case of Chars that checks if the rune slice version of the string argument provided is a valid utf-8 string
//before calling Chars()
func Str(str string) Parsec {
	return func(in ParserInput) PResult {
		if utf8.ValidString(str) {
			return Chars([]rune(str))(in)
		} else {
			return PResult{nil, in}
		}
	}
}

// Many0 will take many as many reps of a parser, even zero 
func (p Parsec) Many0() Parsec {
	return func(in ParserInput) PResult {
		res := PResult{list.New(), in}
		for !res.rem.Empty() {
			out := p(res.rem)
			if out.result == nil {
				return res
			}
			res.result.(*list.List).PushBack(out.result)
			res.rem = out.rem
		}
		return res
	}
}


// Many1 is like Maany0, but must pass at least once 
func (p Parsec) Many1() Parsec {
	return func(in ParserInput) PResult {
		res := PResult{list.New(), in}
		first := p(in)
		if first.result == nil {
			return PResult{nil, in}
		}
		res.result.(*list.List).PushBack(first.result)
		res.rem = first.rem
		for !res.rem.Empty() {
			out := p(res.rem)
			if out.result == nil {
				return res
			}
			res.result.(*list.List).PushBack(out.result)
			res.rem = out.rem
		}
		return res
	}
}

func (p Parsec) Count(n int) Parsec {
	return func(in ParserInput) PResult {
		res := PResult{list.New(), in}
		for i := 0; i < n; i++ {
			out := p(res.rem)
			if out.result == nil {
				return PResult{nil, in}
			}
			res.result.(*list.List).PushBack(out.result)
			res.rem = out.rem
		}
		return res
	}
}

func (p Parsec) Then(sec Parsec) Parsec {
	return func(in ParserInput) PResult {
		res := p(in)
		if res.rem.Empty() || res.result == nil { //firsst parser failed or there's no input left
			return PResult{nil, in}
		}
		res = sec(res.rem)
		return res
	}
}


func  FoldMany0[T any](p Parsec, init func() T, acc func (res, curr T) T ) Parsec {
	return func(in ParserInput) PResult {
		res := init()
		copy := in
		for !copy.Empty(){
			curr := p(copy)
			if curr.result == nil {
				return PResult {res, copy}
			}
			copy = curr.rem
			res = acc(res, curr.result.(T))
		}
		return PResult{res, copy}
	}
}
