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
type Parsec func(in ParserInput) PResult

//Predicate is a function that takes a rune and performs some computation, returning a true/false result
//this result is useful when the predicate is used a a function argument is a higher order function.
//a true result proves that the rune in question satisfies a particular condition
type Predicate func(r rune) bool

//ParserInput specifies two methods.
//The method `Car` returns the next rune in the stream.. an implememter only needs return the first item in its list when Car is called
//Cdr OTOH, while also not changing the internal state of the implementer, returns another copy of the implementer
//without the first part. It works like a `Lisp`
type ParserInput interface {
	Car() rune        //when it is called, it returns the current rune without advancing the index
	Cdr() ParserInput //returns the remainder of the input after the first one has been removed
	Empty() bool
}

//PResult contains two fields. `result` contains the result of the parser. `rem` contains the remaining imput
//if the parser succeeds then `rem` is the remainder of the input after the `matched` runes have been moved out of it
//if the parser fails, the rem contains the input unchanged
type PResult struct {
	Result interface{}
	Rem    ParserInput
	Err    error
}

type Result interface {
	fmt.Stringer
}

func (r *PResult) Errored() (error, bool) {
	return r.Err, r.Err != nil
}

type ParsecErr struct {
	context string
	inner   error
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
	return &ParsecErr{context: "There isn't enough data left for this parser"}
}

var (
	Unmatched  *ParsecErr = &ParsecErr{context: "Parser Unmatched"}
	Incomplete *ParsecErr = &ParsecErr{context: "There isn't enough data left for this parser"}

	// PredicateFailed *ParsecErr = &ParsecErr{context: "The predicate failed without returning anything"}
)

////////SIMPLE PARSERS
// Tag is the simplest parser, it checks if a rune matches the next rune in the input.

func Tag(r rune) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		if r == in.Car() {
			return PResult{r, in.Cdr(), nil}
		}

		return PResult{
			nil, in, UnmatchedErr(),
		}
	}
}

// IsNot is the complete opposite of IsA. It returns the `not(r rune)` that it finds next. If it finds `r`, it fails
func IsNot(r rune) Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		if r == in.Car() {
			return PResult{nil, in, UnmatchedErr()}
		}

		return PResult{in.Car(), in.Cdr(), nil}
	}
}

// CharUTF8 returns a parser which checks if this rune is a valid utf-8 character. thhis character could be any utf-8 symbol
func CharUTF8() Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		curr := in.Car()

		if utf8.ValidRune(curr) {
			return PResult{
				curr, in.Cdr(), nil,
			}
		}

		return PResult{nil, in, UnmatchedErr()}
	}
}

// OneOf returns a perser which checks if the next rune matches one of any given tunes.
// returns a rune
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

// Digit takes only utf-8 encoded runes and ensures they are decimal digits (0-9). It returns a single digit
func Digit() Parsec {
	return func(in ParserInput) PResult {

		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		curr := in.Car()

		//if curr is a unicode number
		if utf8.ValidRune(curr) && unicode.IsDigit(curr) {
			return PResult{
				int(curr - '0'), in.Cdr(), nil,
			}
		}

		//else
		return PResult{nil, in, UnmatchedErr()}
	}
}

// Letter checks if the nexxt rune from the rune stream is a valid utf-8 letter
func Letter() Parsec {
	return func(in ParserInput) PResult {
		if in.Empty() {
			return PResult{nil, in, IncompleteErr()}
		}

		curr := in.Car()
		if utf8.ValidRune(curr) && unicode.IsLetter(curr) {
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

//TakeN eats up `n` number of runes. if it doesnt get up to `n` number of runes, it fails. It retursn a list of runes as Result
func TakeN(n int) Parsec {

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

		for i := 0; i < n; i++ {
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

// returns a string of length n in byte count
func StrN(n int) Parsec {
	return func(in ParserInput) PResult {

		if in.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}

		rem := in //rem needs to have input copied into it because we want to retain the full input, in case of a failure where we return the entire input
		var res strings.Builder
		num := 0
		for {
			if rem.Empty() { //we exhausted the input before taking all we wanted
				return PResult{
					nil,
					in,
					IncompleteErr(),
				}
			}

			//there's more, and we haven't reached our target number
			curr := rem.Car()

			if !utf8.ValidRune(curr) {
				return PResult{
					nil,
					in,
					UnmatchedErr(),
				}
			}
			rem = rem.Cdr()
			currnum, _ := res.WriteRune(curr)
			num += currnum
			if num >= n { // we have reached the specific length on bytes we need
				// comeback: what if num exceeds n?
				return PResult{
					res.String(),
					rem,
					nil,
				}
			}
		}
	}
}

// TakeTill eats runes until a Predicate is satisfied.
//It must take at least one rune for it to be successful
// returns a list of runes
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
					return PResult{
						nil,
						in,
						UnmatchedErr(),
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

// TakeTillIncl is same as TakeTill, but also eats up the byte that satisfies the predicate too
// It doesn't include the last byte in the result, it just consumes it
// It returns a list of runes too
func TakeTillIncl(f Predicate) Parsec {
	return func(in ParserInput) PResult {

		takeTill := TakeTill(f)
		res := takeTill(in)
		if err, didErr := res.Errored(); didErr { // the worker parser returned error
			return PResult{
				Result: nil,
				Rem:    in,
				Err:    err,
			}
		}
		// next Car() should be the rune that astisfies the f predicate
		res.Rem = res.Rem.Cdr()
		return res
	}
}

// TakeWhile keeps eating runes while Pedicate returns true.
// If no rune makes it into the results, `TakeWhile` returns an error
// Returns a slice of runes.
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
		res := []rune{}
		curr := rem.Car()
		for !rem.Empty() && f(curr) {
			res = append(res, curr)
			rem = rem.Cdr()
			curr = rem.Car()
		}

		if len(res) == 0 {
			return PResult{
				nil,
				in,
				UnmatchedErr(),
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
// The Result is the first one, the `match`, because `Termnated` assumes that that is the one we're interested in, and `post` is merely a delimeter.
// It returns the original `match` string if it passes, or nil if it fails
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

		//we need two loops, one for the first string, the second for the other.
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

// Preceded is like `Terminated`, only reversed.
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
// result is an integer
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
		checked, neg := false, false
		for !rem.Empty() {
			//first check if the first rune is a '-', then it'll be negative
			if !checked {
				res := Tag('-')(rem)
				if res.Err == nil {
					neg = true
				}
				rem = res.Rem
				checked = true

			} else {
				res := digs(rem)
				if res.Err == nil {
					if s, ok := res.Result.(rune); ok {
						numStr.WriteRune(s)
						rem = rem.Cdr() // we could use either of `rem.Cdr()` or `res.rem` here because theyre thesame as the Parser `OneOf` eats only the `Car`
					}
				} else {
					e = res.Err
					break
				}

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
		if neg {
			ans = -ans
		}
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
			res := Tag(c)(rem)
			if res.Err != nil { //parser failed to match
				return PResult{
					nil, in, res.Err, //let the error trickle up
				}
			}
			rem = res.Rem
		}

		return PResult{chars, rem, nil}
	}
}

// Str is a special case of Chars that checks if the rune slice version of the string argument provided is a valid utf-8 string
//before calling Chars()
func Str(str string) Parsec {
	return func(in ParserInput) PResult {
		if utf8.ValidString(str) {
			res := Chars([]rune(str))(in)
			// v := reflect.ValueOf(res)

			if chars, ok := res.Result.([]rune); ok {
				return PResult{
					string(chars),
					res.Rem,
					nil,
				}
			} else {
				return PResult{
					nil,
					in,
					&ParsecErr{context: "Could not convert from chars to string"},
				}
			}
		} else {
			return PResult{nil, in, &ParsecErr{context: "String provided is not a valid string"}}
		}
	}
}

// Many0 will drive as many reps of a parser as possible, incl. zero. At the first failure of the parser, it returns without erroring
// result is a slice of results of the parser being repeated
func (p Parsec) Many0() Parsec {
	return func(in ParserInput) PResult {
		resSlice := make([]any, 0)
		res := PResult{resSlice, in, nil}
		for !res.Rem.Empty() {
			out := p(res.Rem)
			if out.Err != nil {
				// still returns a nil error even if the parser eats nothing
				//
				return res
			}
			resSlice = append(resSlice, out.Result)
			res.Result = resSlice
			res.Rem = out.Rem
		}
		return res
	}
}

// Many1 is like `Many0`, but must pass at least once
// result is a slice of results of the parser being repeated

func (p Parsec) Many1() Parsec {
	return func(in ParserInput) PResult {
		resSlice := make([]any, 0)
		res := PResult{resSlice, in, nil}
		//ensuring that at least one succeeds
		first := p(in)
		if e, did := first.Errored(); did {
			res.Err = e.(*ParsecErr)
			return res
		}
		resSlice = append(resSlice, first.Result)
		res.Result = resSlice
		res.Rem = first.Rem

		//now to the loop
		for !res.Rem.Empty() {
			out := p(res.Rem)
			if out.Err != nil {
				return res
			}
			resSlice = append(resSlice, out.Result)
			res.Result = resSlice
			res.Rem = out.Rem
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
			out := p(res.Rem)
			if out.Err != nil {
				return PResult{nil, in, out.Err}
			}
			res.Result.(*list.List).PushBack(out.Result)
			res.Rem = out.Rem
		}
		return res
	}
}

// Then joins two parsers. It discards the result of the first one.
// If the first one suceeds, it calls the second one. If it doesn't it returns an error
// To use `Then`,
func (p Parsec) Then(sec Parsec) Parsec {
	return func(in ParserInput) PResult {
		res := p(in)
		if res.Rem.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}
		if res.Err != nil { //first parser failed or there's no input left
			return PResult{nil, in, UnmatchedErr()}
		}
		res = sec(res.Rem)
		return res
	}
}

// ThenDiscard is like Then, but discards the result of the second parser if it matches.
func (p Parsec) ThenDiscard(sec Parsec) Parsec {
	return func(in ParserInput) PResult {
		res := p(in)
		if res.Rem.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}
		if res.Err != nil { //first parser failed or there's no input left
			return PResult{nil, in, UnmatchedErr()}
		}
		res2 := sec(res.Rem)
		if res2.Err != nil { //first parser failed or there's no input left
			return PResult{nil, in, UnmatchedErr()}
		}
		return res
	}
}

// AndThen joins n parsers.
// If the first one suceeds, it calls the next one. If it doesn't it returns an error
// result is a list of the results of each parser
func (p Parsec) AndThen(secs []Parsec) Parsec {
	return func(in ParserInput) PResult {
		rem := in
		slice := make([]any, 0) // has to be a slice of any type, because we do not know the types of the results of the parsers inputed
		result := PResult{slice, rem, nil}
		out := p(rem)
		if out.Rem.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}
		if out.Err != nil { //first parser failed or there's no input left
			return PResult{nil, in, UnmatchedErr()}
		}
		slice = append(slice, out.Result)
		// result.Result.(*list.List).PushBack(out.Result)
		for _, sec := range secs {
			if out.Rem.Empty() {
				return PResult{
					nil,
					in,
					IncompleteErr(),
				}
			}
			out = sec(out.Rem)
			if _, didErr := out.Errored(); didErr {
				return PResult{list.New(), in, UnmatchedErr()}
			}
			// result.Result.(*list.List).PushBack(out.Result)
			slice = append(slice, out.Result)

		}
		return result
	}
}

// Alt tries a list of parsers and returns the result of the first successful one
func Alt(ps ...Parsec) Parsec {
	return func(in ParserInput) PResult {
		rem := in
		if rem.Empty() {
			return PResult{
				nil,
				in,
				IncompleteErr(),
			}
		}
		for _, p := range ps {
			res := p(rem)
			if _, didErr := res.Errored(); !didErr {
				return res
			}
		}

		return PResult{
			nil,
			in,
			UnmatchedErr(),
		}
	}
}

// Guarded uses `Tag` `TakeTillIncl` to take a list of runes that fill up the space between `left` and `right`
//  result is a lst of runes
func Guarded(left, right rune) Parsec {
	return func(in ParserInput) PResult {
		pre := Tag(left)
		p := pre.Then(TakeTillIncl(func(r rune) bool {
			return r == right
		}))
		return p(in)
	}
}

// GuardedWhile takes two runes as left and right guards, a predicate to specify the conditions each rune
// between the left and the right runes must satisfy
//  the `left` and `right` runes are not parts of the results. they are discarded
// since the internal mechanism of `GuardedWhile` uses `TakeWhile`, the result returned is a slice of runes
func GuardedWhile(left, right rune, p Predicate) Parsec {
	return func(in ParserInput) PResult {
		pre := Tag(left)
		parser := pre.Then(TakeWhile(p))
		res := parser(in)
		if err, didErr := res.Errored(); didErr {
			return PResult{
				Result: nil,
				Rem:    in,
				Err:    err.(*ParsecErr),
			}
		}
		if res.Rem.Empty() {
			return PResult{
				Result: nil,
				Rem:    in,
				Err:    IncompleteErr(),
			}
		}
		result := res.Result
		post := Tag(right)
		res2 := post(res.Rem)
		if err, didErr := res2.Errored(); didErr {
			return PResult{
				Result: nil,
				Rem:    in,
				Err:    err.(*ParsecErr),
			}
		}

		return PResult{
			Result: result,
			Rem:    res.Rem,
			Err:    nil,
		}
	}
}

type Callback func(any, Parsec) PResult

func FoldMany0[T any](p Parsec, init func() T, accFunc func(res, curr T) T) Parsec {
	return func(in ParserInput) PResult {
		res := init() //T
		copy := in
		for !copy.Empty() {
			curr := p(copy)
			if curr.Err != nil {
				return PResult{res, copy, nil}
			}
			copy = curr.Rem
			res = accFunc(res, curr.Result.(T))
		}
		return PResult{res, copy, nil}
	}
}

func FoldMany1[T any](p Parsec, init func() T, accFunc func(res, curr T) T) Parsec {
	return func(in ParserInput) PResult {
		res := init() //T
		copy := in
		n := 0
		for !copy.Empty() {
			curr := p(copy)
			if curr.Err != nil {
				if n < 1 {
					return PResult{nil, in, UnmatchedErr()} //parser failed without accumulating anything
				} else {
					return PResult{res, copy, nil} //parser failed after accumutating at least once
				}

			}
			copy = curr.Rem
			res = accFunc(res, curr.Result.(T))
			n++
		}
		return PResult{res, copy, nil}
	}
}
