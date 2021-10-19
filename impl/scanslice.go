package impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	sqldb "github.com/domonda/go-sqldb"
)

// ScanRowsAsSlice scans all srcRows as slice into dest.
// The rows must either have only one column compatible with the element type of the slice,
// or if multiple columns are returned then the slice element type must me a struct or struction pointer
// so that every column maps on exactly one struct field using structFieldNamer.
// In case of single column rows, nil must be passed for structFieldNamer.
// ScanRowsAsSlice calls srcRows.Close().
func ScanRowsAsSlice(ctx context.Context, srcRows Rows, dest interface{}, structFieldNamer sqldb.StructFieldNamer) error {
	defer srcRows.Close()

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("scan dest is not a pointer but %s", destVal.Type())
	}
	if destVal.IsNil() {
		return errors.New("scan dest is nil")
	}
	slice := destVal.Elem()
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("scan dest is not pointer to slice but %s", destVal.Type())
	}
	sliceElemType := slice.Type().Elem()

	newSlice := reflect.MakeSlice(slice.Type(), 0, 32)

	for srcRows.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		newSlice = reflect.Append(newSlice, reflect.Zero(sliceElemType))
		target := newSlice.Index(newSlice.Len() - 1).Addr()
		if structFieldNamer != nil {
			err := ScanStruct(srcRows, target.Interface(), structFieldNamer, nil, nil)
			if err != nil {
				return err
			}
		} else {
			err := srcRows.Scan(target.Interface())
			if err != nil {
				return err
			}
		}
	}
	if srcRows.Err() != nil {
		return srcRows.Err()
	}

	// Assign newSlice if there were no errors
	if newSlice.Len() == 0 {
		slice.SetLen(0)
	} else {
		slice.Set(newSlice)
	}

	return nil
}

// SplitArray splits an SQL or JSON array into its top level elements.
// Returns nil in case of an empty array ("{}" or "[]").
func SplitArray(array string) ([]string, error) {
	if len(array) < 2 {
		return nil, fmt.Errorf("%q is too short for an array", array)
	}
	first := array[0]
	last := array[len(array)-1]
	isJSON := first == '[' && last == ']'
	isSQL := first == '{' && last == '}'
	if !isJSON && !isSQL {
		return nil, fmt.Errorf("%q is not a SQL or JSON array", array)
	}
	inner := strings.TrimSpace(array[1 : len(array)-1])
	if inner == "" {
		return nil, nil
	}
	var (
		elems        []string
		objectDepth  = 0
		bracketDepth = 0
		elemStart    = 0
		rLast        rune
		withinQuote  rune
	)
	for i, r := range inner {
		if withinQuote == 0 {
			switch r {
			case ',':
				if objectDepth == 0 && bracketDepth == 0 {
					elems = append(elems, strings.TrimSpace(inner[elemStart:i]))
					elemStart = i + 1
				}

			case '{':
				objectDepth++

			case '}':
				objectDepth--
				if objectDepth < 0 {
					return nil, fmt.Errorf("array %q has too many '}'", array)
				}

			case '[':
				bracketDepth++

			case ']':
				bracketDepth--
				if bracketDepth < 0 {
					return nil, fmt.Errorf("array %q has too many ']'", array)
				}

			case '"':
				// Begin JSON string
				withinQuote = r

			case '\'':
				// Begin SQL string
				withinQuote = r
			}
		} else {
			// withinQuote != 0
			switch withinQuote {
			case '\'':
				if r == '\'' && rLast != '\'' {
					// End of SQL quote because ' was not escapded as ''
					withinQuote = 0
				}
			case '"':
				if r == '"' && rLast != '\\' {
					// End of JSON quote because " was not escapded as \"
					withinQuote = 0
				}
			}
		}

		rLast = r
	}

	if objectDepth != 0 {
		return nil, fmt.Errorf("array %q has not enough '}'", array)
	}
	if bracketDepth != 0 {
		return nil, fmt.Errorf("array %q has not enough ']'", array)
	}
	if withinQuote != 0 {
		return nil, fmt.Errorf("array %q has an unclosed '%s' quote", array, string(withinQuote))
	}

	// Rameining element after begin and separators
	if elemStart < len(inner) {
		elems = append(elems, strings.TrimSpace(inner[elemStart:]))
	}

	return elems, nil
}

type SliceScanner struct {
	destSlice reflect.Value
}

func WrapWithSliceScanner(destPtr interface{}) interface{} {
	v := reflect.ValueOf(destPtr)
	if v.Elem().Kind() != reflect.Slice || v.Type().Implements(typeOfSQLScanner) {
		return destPtr
	}
	return SliceScanner{destSlice: v.Elem()}
}

// Scan implements the sql.Scanner interface.
func (a *SliceScanner) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return a.scanString(string(src))
	case string:
		return a.scanString(src)
	default:
		return fmt.Errorf("can't scan %T as slice", src)
	}
}

func (a *SliceScanner) scanString(src string) error {
	elems, err := SplitArray(src)
	if err != nil {
		return err
	}
	if len(elems) == 0 {
		a.destSlice.Set(reflect.Zero(a.destSlice.Type()))
		return nil
	}
	elemType := a.destSlice.Type().Elem()
	newSlice := reflect.MakeSlice(elemType, len(elems), len(elems))
	if reflect.PtrTo(elemType).Implements(typeOfSQLScanner) {
		for i, elem := range elems {
			err = newSlice.Index(i).Addr().Interface().(sql.Scanner).Scan(elem)
			if err != nil {
				return fmt.Errorf("can't scan %q as element %d of slice %s because of %w", elem, i, elemType, err)
			}
		}
	} else {
		for i, elem := range elems {
			err = ScanValue(elem, newSlice.Index(i))
			if err != nil {
				return fmt.Errorf("can't scan %q as element %d of slice %s because of %w", elem, i, elemType, err)
			}
		}
	}
	a.destSlice.Set(newSlice)
	return nil
}

func ScanValue(src interface{}, dest reflect.Value) error {
	if dest.Kind() == reflect.Interface {
		if src != nil {
			dest.Set(reflect.ValueOf(src))
		} else {
			dest.Set(reflect.Zero(dest.Type()))
		}
		return nil
	}

	if dest.Addr().Type().Implements(typeOfSQLScanner) {
		return dest.Addr().Interface().(sql.Scanner).Scan(src)
	}

	switch x := src.(type) {
	case int64:
		switch dest.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			dest.SetInt(x)
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dest.SetUint(uint64(x))
			return nil
		case reflect.Float32, reflect.Float64:
			dest.SetFloat(float64(x))
			return nil
		}

	case float64:
		switch dest.Kind() {
		case reflect.Float32, reflect.Float64:
			dest.SetFloat(x)
			return nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			dest.SetInt(int64(x))
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dest.SetUint(uint64(x))
			return nil
		}

	case bool:
		dest.SetBool(x)
		return nil

	case []byte:
		switch {
		case dest.Kind() == reflect.String:
			dest.SetString(string(x))
			return nil
		case dest.Kind() == reflect.Slice && dest.Type().Elem().Kind() == reflect.Uint8:
			dest.Set(reflect.ValueOf(x))
			return nil
		}

	case string:
		switch {
		case dest.Kind() == reflect.String:
			dest.SetString(x)
			return nil
		case dest.Kind() == reflect.Slice && dest.Type().Elem().Kind() == reflect.Uint8:
			dest.Set(reflect.ValueOf([]byte(x)))
			return nil
		}

	case time.Time:
		if srcVal := reflect.ValueOf(src); srcVal.Type().AssignableTo(dest.Type()) {
			dest.Set(srcVal)
			return nil
		}

	case nil:
		switch dest.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map:
			dest.Set(reflect.Zero(dest.Type()))
			return nil
		}
	}

	return fmt.Errorf("can't scan %#v as %s", src, dest.Type())
}
