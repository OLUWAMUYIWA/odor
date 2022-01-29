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

//Parsec is a basic parser function. It takes an imput and returns PResult as Result.
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
	Result interface{}
	rem    ParserInput
	err error
}

func (r *PResult) Errored() (error, bool) {
	return r.err, r.err  != nil
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

func UnmatchedErr() *ParsecErr {
	return &ParsecErr{context: "Parser Unmatched"}
}

func IncompleteErr() *ParsecErr {
	return &ParsecErr{context: "There isn't enough data left fot this parser"}
}

var  (
	Unmatched *ParsecErr = &ParsecErr{context: "Parser Unmatched"}
	Incomplete *ParsecErr = &ParsecErr{context: "There isn't enough data left fot this parser"}
	PredicateFailed *ParsecErr=  &ParsecErr{context: "The predicate failed without return ing anything"}
)

////////SIMPLE PARSERS
// IsA is the simplest parser, it checks if a rune matches the next rune in the input.
func IsA(r rune) Parsec {
	return func (in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		if r == in.Car() { 
			return PResult{r,  in.Cdr(), nil}
		}

		return PResult{
			nil, in, UnmatchedErr(),
		}
	}
}


// IsNot is the complete opposite of IsA.
func IsNot(r rune) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		if r == in.Car() {
			return PResult{nil, in, nil}
		}

		return PResult{r, in.Cdr(), UnmatchedErr()}
	}
}

// CharUTF* returns a parser which checks if this rune is a valid utf-8 character. thhis character could be any utf-8 symbol
func CharUTF8(c rune) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		curr := in.Car()
		
		if  utf8.ValidRune(curr) {
			return PResult{
				curr, in.Cdr(), nil,
			}
		}

		return PResult{nil, in, UnmatchedErr()}
	}
}


//OneOf returns a perser which checks if the next rune matches one of any given tunes
func OneOf(any []rune) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}
		curr := in.Car()
		for _, r := range any {
			if curr == r {
				return PResult{
					r,
					in.Cdr(),
					nil,
				}
			}
		}

		//no match found
		return PResult{
			nil,
			in,
			UnmatchedErr(),
		}
	}
}

// Digit takes only utf-8 encoded runes and ensures they are decimal digits (0-9)
func Digit() Parsec {
	return func(in ParserInput) PResult {
		
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		curr := in.Car()

		//if curr is a unicode number
		if utf8.ValidRune(curr) && unicode.IsDigit(curr) {
			return PResult{
				curr, in.Cdr(), nil,
			}
		}

		//else
		return PResult{nil, in, UnmatchedErr()}
	}
}

// Letter takes only utf-8 encoded runes and ensures they are letters
func Letter(c rune) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		curr := in.Car()
		if  utf8.ValidRune(curr) && unicode.IsLetter(curr) {
			return PResult{
				curr,
				in.Cdr(),
				nil,
			}
		}
		return PResult{
			nil,
			in,
			UnmatchedErr(),
		}
	}
}

/////REPETITIONS

//Take eats up `n` number of runes. if it doesnt get up to `n` number of runes, it fails. It retursn a list of runes as Result
func Take(n int) Parsec {

	return func(in ParserInput) PResult {

		if in.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}

		rem := in //rem needs to have input copied into it because we want to retain the full input, in case of a failure where we return the entire input
		res := list.New()

		for i := 0; i < n-1; i++ { 
			if rem.Empty() { //we exhausted the input before taking all we wanted
				return PResult{
					nil,
					in,
					IncompleteErr(),
				}
			} else { //there's more, and we haven't reached our target number
				res.PushBack(rem.Car())
				rem = rem.Cdr()
			}

		}

		if res.Len() < n { //doublecheck
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}

		return PResult{
			res,
			rem,
			nil,
		}
	}
}

// TakeTill eats runes until a Predicate is satisfied. It must take at least one rune for it to be successful
func TakeTill(f Predicate) Parsec {
	return func(in ParserInput) PResult {

		if in.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}

		rem := in
		res := list.New()
		curr := rem.Car()

		for !rem.Empty() {

			if f(curr) { //if the predicate returns true, we're done
				if res.Len() == 0 { //we gained nothing from the parser
					return PResult {
						nil,
						in,
						PredicateFailed,
					}
				}

				return PResult{
					res,
					rem,
					nil,
				}
			}

			res.PushBack(curr)
			rem = rem.Cdr()
			curr = rem.Car()
		}

		
		return PResult{
			res,
			rem,
			nil,
		}
	}
}

// TakeWhile keeps eating runes while Pedicate returns true. Returns a list of runes. 
// If no rune makes it into the results, `TakeWhile` returns an error
func TakeWhile(f Predicate) Parsec {

	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{
				nil,
				in,
				UnmatchedErr(),
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
				PredicateFailed,
			}
		}

		return PResult{
			res,
			rem,
			nil,
		}
	}
}

// Terminated asks if the first argument `match` is `followed` immediately by the second one `post`
// Terminated takes `strings` and not runes. This makes it quite easier to use with string-based protocols
// The Result is the first one, the `match`, because `Termnated` assumes that that is the one we're interested in. 
func Terminated(match, post string) Parsec {

	return func(in ParserInput) PResult {

		if in.Empty() {
			return PResult{
				nil,
				in,
				Incomplete,
			}
		}

		rem := in
		matchRunes, postRunes := []rune(match), []rune(post) //create rune slices from the strings
		
		//we need two loops, ne for the first string, the second for the other.
		//If we fail anywhere in running through the two loops, we fail out immediately

		for _, r := range matchRunes {

			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
					IncompleteErr(),
				}
			}

			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
					UnmatchedErr(),
				}
			}

			rem = rem.Cdr()
			
		}

		//second loop
		for _, r := range postRunes {

			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
					IncompleteErr(),
				}
			}

			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
					UnmatchedErr(),
				}
			}

			rem = rem.Cdr()
			
		}

		return PResult{
			match,
			rem,
			nil,
		}
	}
}

//Preceded is like `Terminated`, only reversed. 
// It asks if `match` is preceded by `pre`, and returns `match` as Result if it does, and a nil Result and error if it doesn't
func Preceded(match, pre string) Parsec {
	return func(in ParserInput) PResult {

		if in.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}
		rem := in
		matchRunes, preRunes := []rune(match), []rune(pre) //create rune slices from the strings

		//first loop, for the `pre` argument
		for _, r := range preRunes {
			
			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
					IncompleteErr(),
				}
			}

			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
					UnmatchedErr(),
				}
			}
			rem = rem.Cdr()
		}

		//second loop, for the `match` argument
		for _, r := range matchRunes {
			
			if rem.Empty() { //input empties without us eating all the runes we want
				return PResult{
					nil,
					in,
					IncompleteErr(),
				}
			}

			curr := rem.Car()
			if curr != r {
				return PResult{
					nil,
					in,
					UnmatchedErr(),
				}
			}
			rem = rem.Cdr()
			
		}

		return PResult{
			match,
			rem,
			nil,
		}
	}
}

// Number asks if it can obtain a contiguous set of digits from the input stream
func Number() Parsec {
	return func(in ParserInput) PResult {
		
		if in.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}

		var numStr strings.Builder
		numbers := []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
		digs := OneOf(numbers)
		rem := in
		var e error
		for !rem.Empty() {
			res := digs(rem)
			if res.err == nil {
				if s, ok := res.Result.(rune); ok {
					numStr.WriteRune(s)
					rem = rem.Cdr() // we could use either of `rem.Cdr()` or `res.rem` here because theyre thesame as the Parser `OneOf` eats only the `Car`
				}
			} else {
				e = res.err
				break
			}
		}

		//no digit was found, so no number
		if numStr.String() == "" {
			return PResult{
				nil,
				in,
				e,
			}
		}

		ans, _ := strconv.Atoi(numStr.String()) // there can never be an error.
		return PResult{
			ans,
			rem,
			nil,
		}
	}
}

// Chars asks if a stream of input matches the characters in the rune slice provided
// if it doesn't, the entire input is returned unchanged, but with a nil Result
func Chars(chars []rune) Parsec {
	return func(in ParserInput) PResult {

		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		rem := in //remainder is first the entire input
		
		for _, c := range chars {
			res := CharUTF8(c)(rem)
			if res.err != nil { //parser failed to match
				return PResult{
					nil, in, res. err, //let the error trickle up
				}
			}
			rem = res.rem
		}

		return PResult{chars, rem, nil}
	}
}

// Str is a special case of Chars that checks if the rune slice version of the string argument provided is a valid utf-8 string
//before calling Chars()
func Str(str string) Parsec {
	return func(in ParserInput) PResult {
		if utf8.ValidString(str) {
			return Chars([]rune(str))(in)
		} else {
			return PResult{nil, in, &ParsecErr{context: "String provided is not a valid string"}}
		}
	}
}

// Many0 will take many as many reps of a parser, even zero. At the first failure of the parser, it returns witout erroring
func (p Parsec) Many0() Parsec {
	return func(in ParserInput) PResult {
		res := PResult{list.New(), in, nil}
		for !res.rem.Empty() {
			out := p(res.rem)
			if out.err != nil {
				return res
			}
			res.Result.(*list.List).PushBack(out.Result) //coerce the `interface{}` Result value into a `*list.List` value
			res.rem = out.rem
		}
		return res
	}
}


// Many1 is like `Many0`, but must pass at least once 
func (p Parsec) Many1() Parsec {
	return func(in ParserInput) PResult {
		res := PResult{list.New(), in, nil}

		//ensuring that at least one succeeds
		first := p(in)
		if first.err != nil {
			return PResult{nil, in, first.err} //if it doesn't suceed, the Result contains the 
		}
		res.Result.(*list.List).PushBack(first.Result)
		res.rem = first.rem

		//now to the loop
		for !res.rem.Empty() {
			out := p(res.rem)
			if out.err != nil {
				return res
			}
			res.Result.(*list.List).PushBack(out.Result)
			res.rem = out.rem
		}
		return res
	}
}

// Count applies the mother parser `n` times, if the parser fails before the n'th time, `Count` fails too. It retrns a list 
// of the original parser's results

func (p Parsec) Count(n int) Parsec {
	return func(in ParserInput) PResult {
		res := PResult{list.New(), in, nil}
		for i := 0; i < n; i++ {
			out := p(res.rem)
			if out.err != nil {
				return PResult{nil, in, out.err}
			}
			res.Result.(*list.List).PushBack(out.Result)
			res.rem = out.rem
		}
		return res
	}
}

// Then jins two parsers. If the first one suceeds, it calls the second one. If it doesn't it returns an error
func (p Parsec) Then(sec Parsec) Parsec {
	return func(in ParserInput) PResult {
		res := p(in)
		if res.rem.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}
		if res.err != nil { //firsst parser failed or there's no input left
			return PResult{nil, in, UnmatchedErr()}
		}
		res = sec(res.rem)
		return res
	}
}




func  FoldMany0[T any](p Parsec, init func() T, accFunc func (res, curr T) T ) Parsec {
	return func(in ParserInput) PResult {
		res := init()  //T
		copy := in
		for !copy.Empty(){
			curr := p(copy)
			if curr.err != nil{
				return PResult {res, copy, nil}
			}
			copy = curr.rem
			res = accFunc(res, curr.Result.(T))
		}
		return PResult{res, copy, nil}
	}
}

func  FoldMany1[T any](p Parsec, init func() T, accFunc func (res, curr T) T ) Parsec {
	return func(in ParserInput) PResult {
		res := init()  //T
		copy := in
		n := 0
		for !copy.Empty(){
			curr := p(copy)
			if curr.err != nil {
				if n < 1 {
					return PResult {nil, in, UnmatchedErr()} //parser failed without accumulating anything
				} else {
					return PResult {res, copy, nil} //parser failed after accumutating at least once
				}
				
			}
			copy = curr.rem
			res = accFunc(res, curr.Result.(T))
			n++
		}
		return PResult{res, copy, nil}
	}
}
