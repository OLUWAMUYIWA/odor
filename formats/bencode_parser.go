package formats

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"

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

func BStr() {
	parsec.Number()
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