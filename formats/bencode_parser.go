package formats

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/OLUWAMUYIWA/odor/parsec"
)


func BStr() {
	parsec.Number()
}

type Decoder struct {
	r io.Reader
}

func (d *Decoder) Parse(structure any) error {
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