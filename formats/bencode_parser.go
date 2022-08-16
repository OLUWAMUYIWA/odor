package formats

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"unicode"

	"github.com/OLUWAMUYIWA/odor/parsec"
)

type BencInput struct {
	R *bufio.Reader
}

func NewBencInput(r io.Reader) *BencInput {
	return &BencInput{
		R: bufio.NewReader(r),
	}
}

// basically a peek but returned. the input must not be changed after a Car
func (b *BencInput) Car() byte {
	s, _ := b.R.Peek(1)
	return s[0]
}

// read what was last read+unread by Car and drop
func (b *BencInput) Cdr() parsec.ParserInput {

	b.R.ReadByte()
	return b
}

// we say that any error here is due to EOF, but that's unsound. There
// might be an error while interpreting the rune. //comeback
func (b *BencInput) Empty() bool {
	_, err := b.R.Peek(1)
	if err != nil {
		return true
	}
	return false
}

// only used for debugging
func (b BencInput) String() string {
	s, _ := b.R.ReadString('0')
	return s
}

func BencStr() parsec.Parsec {
	return func(in parsec.ParserInput) parsec.PResult {
		rem := in
		if in.Empty() {
			return parsec.PResult{
				Result: nil,
				Rem:    rem,
				Err:    parsec.IncompleteErr(),
			}
		}
		numRes := parsec.Number()(rem)
		if err, didErr := numRes.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err.(*(parsec.ParsecErr))}
		}
		rem = numRes.Rem
		resColon := parsec.Tag(':')(rem)

		if err, didErr := resColon.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err.(*(parsec.ParsecErr))}
		}
		num := numRes.Result.(int)
		rem = resColon.Rem
		if num == 0 { // the case of the empty string: `0:`
			return parsec.PResult{
				Result: "",
				Rem:    rem,
				Err:    nil,
			}
		}

		// non-empty string
		res := parsec.StrN(num)(rem)
		if err, didErr := res.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err.(*(parsec.ParsecErr))}
		}

		return res
	}
}

func BencInt() parsec.Parsec {
	l, r := byte('i'), byte('e')
	guardedInt := parsec.GuardedWhile(l, r, func(r byte) bool { return unicode.IsDigit(rune(r)) })
	return func(in parsec.ParserInput) parsec.PResult {
		res := guardedInt(in)
		// the internal TaeWhile used to implement GuardedWhile returns a slice of runes as result
		// l := res.Result.(*list.List)
		// var digits []rune
		// for e := l.Front(); e != nil; e = e.Next() {
		// 	digits = append(digits, e.Value.(rune))
		// }
		digits, ok := res.Result.([]byte)
		if !ok {
			return parsec.PResult{
				Result: nil,
				Rem:    in,
				Err:    parsec.UnmatchedErr(),
			}
		}
		if _, did := res.Errored(); did {
			return res
		}
		num, _ := strconv.ParseInt(string(digits), 10, 0)
		res.Result = int(num)
		return res
	}
}

// BencList:  possible return types: slice of strings, slice of ints, slice of maps, slice of slices of any of th above
func BencList() parsec.Parsec {
	pre := parsec.Tag('l')
	last := parsec.Tag('e')
	// since empty  lists exist, we use Many0
	manyStr := BencStr().Many0().ThenDiscard(last)
	manyInt := BencInt().Many0().ThenDiscard(last)
	benDict := BenDict().Many0().ThenDiscard(last)
	return func(in parsec.ParserInput) parsec.PResult {
		res := pre(in)
		if err, didErr := res.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err}
		}
		rem := res.Rem

		// fast path: an alternative between a list of integers, strings, or dictionaries
		// `Alt` is not particularly wasteful because the kind of encoding we're dealing with quickly exits if the type were trying is wrong
		res = parsec.Alt(manyInt, manyStr, benDict)(rem)
		if _, didErr := res.Errored(); !didErr {
			return res // return the result as is, its perfect
		}

		//might be a list of lists
		l := [][]any{}
		listsRes := parsec.PResult{
			Result: l,
			Rem:    in,
			Err:    nil,
		}

		for {
			res = BencList()(rem) // use the same input as what was returned in the pre stage
			if err, didErr := res.Errored(); didErr {
				return parsec.PResult{
					Result: nil,
					Rem:    in,
					Err:    err,
				}
			}
			currRes, ok := res.Result.([]any)
			if !ok {
				return parsec.PResult{
					Result: nil,
					Rem:    in,
					Err:    parsec.UnmatchedErr(),
				}
			}
			l = append(l, currRes) // would be a slice of any of the possible types a list can contain
			rem = res.Rem
			listsRes.Rem = res.Rem

			// now check whether to exit the loop or not
			end := last(rem)
			if err, didErr := end.Errored(); didErr && errors.Is(err, parsec.UnmatchedErr()) {
				// UnmatchedErr means the rune does not match, meaning that we're not done yet, but theres more data to go
				continue
			} else if errors.Is(err, parsec.IncompleteErr()) { // there's no more data to eat, therefore, the list is open-ended: incomplete
				return parsec.PResult{
					Result: nil,
					Rem:    in,
					Err:    parsec.IncompleteErr(),
				}
			} else { // we have reached the end: the rune `e`, which ends the list matches
				return listsRes
			}
		}

	}

}

func BenDict() parsec.Parsec {
	prefix := parsec.Tag('d')
	suffix := parsec.Tag('e')
	key := BencStr()
	str := BencStr()
	num := BencInt()
	list := BencList()
	nonDicts := parsec.Alt(num, str, list)
	return func(in parsec.ParserInput) parsec.PResult {
		dict := map[string]any{}

		// first check the prefix for dictionaries
		res := prefix(in)
		if err, didErr := res.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err}
		}
		rem := res.Rem
		mapRes := parsec.PResult{
			Result: dict,
			Rem:    in,
			Err:    nil,
		}
		for {
			keyRes := key(rem)
			if _, didErr := keyRes.Errored(); didErr { // comeback: coulld be the end of the dict
				break
			}
			rem = keyRes.Rem
			k := keyRes.Result.(string)
			v := nonDicts(rem)
			rem = v.Rem
			if err, didErr := v.Errored(); didErr { // i.e. none of these `nonDicts` parsers passed
				if errors.Is(err, parsec.UnmatchedErr()) { // what we have here should be a dict inside a dict
					res = BenDict()(rem)
					if err, didErr := res.Errored(); didErr { // ow, we have no other type to match, we have erred
						return parsec.PResult{
							Result: nil,
							Rem:    in,
							Err:    err,
						}
					} else { // in this case, we were right. it is a dictionary
						dict[k] = res.Result
					}
				} else if errors.Is(err, parsec.IncompleteErr()) { // we have an error here. a key without a value
					return parsec.PResult{
						Result: nil,
						Rem:    in,
						Err:    parsec.IncompleteErr(),
					}
				} // there's no `else` because we have only two kinds of errors

			} else { // the value matches with the `nonDicts` parser. i.e. a fast path
				// add the value to the cache
				dict[k] = v
			}

			// in any case, we then check if we're done
			end := suffix(rem)
			if err, didErr := end.Errored(); didErr && errors.Is(err, parsec.UnmatchedErr()) { // not yet done
				continue
			} else if errors.Is(err, parsec.IncompleteErr()) {
				return parsec.PResult{ // certainly has to be an error. the dictionary is open-ended, not terminated by a `e`
					Result: nil,
					Rem:    in,
					Err:    parsec.IncompleteErr(),
				}
			} else { // it matches the end
				mapRes.Rem = end.Rem // the remainder becomes the remainder after the terminator has been taken
				mapRes.Result = dict // just to be double-sure. comeback: may not be necessary
				break                // we break ou of the loop
			}
		}

		return mapRes

	}
}

type BencDecoder struct {
	r io.Reader
}

func NewBencDecoder(r io.Reader) *BencDecoder {
	return &BencDecoder{
		r: r,
	}
}

func ParseBenc(in parsec.ParserInput, m map[string]interface{}) error {
	return nil
}

func (d *BencDecoder) Decode(structure any) error {

	m := make(map[string]interface{})
	r := bufio.NewReader(d.r)
	in := &BencInput{r}
	err := ParseBenc(in, m)
	if err != nil {
		return err
	}

	ty := reflect.TypeOf(structure)

	if ty.Kind() != reflect.Pointer {
		return errors.New("Cannot parse into non-pointer")
	}

	// (reflect.Type).Elem() checks the value (either pointer or interface) and returns the value it points to
	elem := ty.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("Cannot parse into non-struct")
	}

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		if field.Tag == "" {
			continue
		}
		tagValue := field.Tag.Get("benc")

		if tagValue == "" {
			continue
		}
		kind := field.Type.Kind()

		switch kind {
		case reflect.String:
			{

			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			{

			}

		case reflect.Slice:
			{

			}

		case reflect.Map:
			{

			}

		default:
			{
				return fmt.Errorf("Data Type Not supported")
			}

		}
	}
	return nil
}
