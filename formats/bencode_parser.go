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
	r *bufio.Reader
}

// basically a peek but returned. the input must not be changed after a Car
func (b *BencInput) Car() rune {
	r,_,_ := b.r.ReadRune()
	b.r.UnreadRune()
	return r
}

// read what was last read+unread by Car and drop
func (b *BencInput) Cdr() parsec.ParserInput {
	_,_,_ = b.r.ReadRune()
	return b
}


// we say that any error here is due to EOF, but that's unsound. There 
// might be an error while interpreting the rune. //comeback
func (b *BencInput) Empty() bool {
	_, _,e := b.r.ReadRune()

	if e != nil {
		return false
	}

	b.r.UnreadRune()
	return true
}

func BencStr() parsec.Parsec{
	return func(in parsec.ParserInput) parsec.PResult {
		rem := in
		if in.Empty() {
			return parsec.PResult{
				Result: nil,
				Rem: in,
				Err: parsec.IncompleteErr(),
			}
		}
		numRes := parsec.Number()(rem)
		if err, didErr := numRes.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err.(*(parsec.ParsecErr)) }
		} 
		rem = numRes.Rem
		resColon := parsec.Tag(':')(rem)
		
		if err, didErr := resColon.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err.(*(parsec.ParsecErr)) }
		} 
		rem = resColon.Rem
		res := parsec.StrN(numRes.Result.(int))(rem)
		if err, didErr := res.Errored(); didErr {
			return parsec.PResult{Result: nil, Rem: in, Err: err.(*(parsec.ParsecErr)) }
		} 

		return res
	}
}


func BencInt() parsec.Parsec {
	guardedInt := parsec.GuardedWhile('i', 'e', func(r rune) bool {
		return unicode.IsDigit(r)
	})
	return func(in parsec.ParserInput) parsec.PResult {
		res := guardedInt(in)
		// the internal TaeWhile used to implement GuardedWhile returns a slice of runes as result
		digits := res.Result.([]rune)
		digitsStr := string(digits)
		num, _ := strconv.Atoi(digitsStr)
		res.Result = num
		return res
	}
}

func BencList() parsec.Parsec {
	pre := parsec.Tag('l')
	last := parsec.Tag('e')
	manyStr := BencStr().Many0().ThenDiscard(last) //comeback the case of empty string
	manyInt := BencInt().Many0().ThenDiscard(last)
	benDict := BenDict().Many0().ThenDiscard(last)
	return func(in parsec.ParserInput) parsec.PResult {
		res := pre(in)
		if err, didErr := res.Errored(); didErr {
			return parsec.PResult{nil, in, err.(*parsec.ParsecErr)}
		}
		res = parsec.Alt(manyInt, manyStr, benDict)(res.Rem)
		if _, didErr := res.Errored(); !didErr {
			return res
		}
		//might be a list of lists
		l := []any{}
		listsRes := parsec.PResult{l, in, nil}

		for {
			res = BencList()(res.Rem)
			if err, didErr := res.Errored(); didErr {
				return parsec.PResult{
					nil,
					in,
					err.(*parsec.ParsecErr),
				}
			}
			l = append(l, res.Result)
			listsRes.Rem = res.Rem
		}
		

		//return parsec.PResult{nil, in, err.(*parsec.ParsecErr)}
		return res
	}

}

func BenDict() parsec.Parsec {
	pre := parsec.Tag('d')
	last := parsec.Tag('e')
	key := BencStr()
	str := BencStr()
	num := BencInt()
	list := BencList()
	nonDicts := parsec.Alt(num, str, list)
	return func(in parsec.ParserInput) parsec.PResult {
		dict := map[string]any{}
		res := pre(in)
		if err, didErr := res.Errored(); didErr {
			return parsec.PResult{nil, in, err.(*parsec.ParsecErr)}
		}

		for {
			keyRes := key(in)
			if _, didErr := keyRes.Errored(); didErr { // i.e. none of these parsers passed
				break
			}
			k := keyRes.Result.(string)
			v := nonDicts(in)
			if _, didErr := v.Errored(); didErr { // i.e. none of these parsers passed
				// we then try to parse it as a dictionary
				v = BenDict()(keyRes.Rem)

				if _, didErr := v.Errored(); didErr{
					break
				}
			}
			dict[k] = v
		}
		
	}
}




type Decoder struct {
	r io.Reader
}


func ParseBenc(in parsec.ParserInput, m map[string]interface{}) error {
	return nil
}

func (d *Decoder) Decode(structure any) error {

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
        case reflect.String: {

        }

    	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64: {

    	}

    	case reflect.Slice: {

    	}

    	case reflect.Map: {

    	} 

    	default: {
    		return fmt.Errorf("Data Type Not supported")
    	}

        }
	}
	return nil
}