package formats

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"golang.org/x/text/cases"
)


type BencEncoder struct {
	wtr io.Writer
}

func NewBencoder(wtr io.Writer) *BencEncoder {
	json.NewEncoder(wtr)
	return &BencEncoder{
		wtr: wtr,
	}
}

// Bencoder expcets to be fed structs, it encodes it into a writer stream
//comeback: is it necessary to restrict it to structs
func (b *BencEncoder) Encode(v any) error {
	val := reflect.ValueOf(v).Elem() // this allows us to be able to take both structs and pointer to structs
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("W expect to encode structs")
	}
	return marshall(val, b.wtr)
}


// marshall is a subroutine used by `Encode` to do the actual marshalling of 
func marshall(v reflect.Value, w io.Writer) error {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64: {
		io.WriteString(w, "i")
		i := v.Int()
		io.WriteString(w, strconv.FormatInt(i, 10))
		io.WriteString(w, "e")
	}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64: {
		io.WriteString(w, "i")
		i := v.Uint()
		io.WriteString(w, strconv.FormatUint(i, 10))
		io.WriteString(w, "e")
	} 
	case reflect.String: {
		s := v.String()
		io.WriteString(w, (strconv.FormatInt(int64(len(s)), 10)))
		io.WriteString(w, fmt.Sprintf(":%s", s))
	} 
	case reflect.Slice: {
		io.WriteString(w, "l")
		for i := 0; i < v.Len(); i++ {
			if err := marshall(v.Index(i), w); err != nil {
					return err
			}
		}
		// old way i tried
		// if slInt, ok := v.Interface().([]int); ok {
		// 	for _, i := range slInt {
		// 		if err := marshall(reflect.ValueOf(i), w); err != nil {
		// 			return err
		// 		}
		// 	}
		// } 
		io.WriteString(w, "e ")

	}
	case reflect.Map: {
		io.WriteString(w, "d")
		// first check if the first key is a string
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			if err := marshall(value, w); err != nil {
					return err
			}
			// switch t := value.Interface().(type) {
			// case int, string, []int, []string: {
			// 	if err := marshall(value, w); err != nil {
			// 		return err
			// 	}
			// } 
			}
		}
		io.WriteString(w, "e")
	}
	case reflect.Struct: {
		num := v.NumField()
		for i := 0; i < num; i++ {
			f := v.Field(i)
			switch f.Kind(): { // only our supported types are allowed here
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.String, reflect.Slice, reflect.Map: {
				if err := marshall(f, w); err != nil {
					return err
				}
			}
			case default: {
				return fmt.Errorf("Unsupported type!")				
			}
			}
		}
	} 
	default: {
		return fmt.Errorf("Unsupported type!")
	}
	}
	return nil
}
