package impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-types/nullable"
)

// ScanRowsAsSlice scans all srcRows as slice into dest.
// The rows must either have only one column compatible with the element type of the slice,
// or if multiple columns are returned then the slice element type must me a struct or struction pointer
// so that every column maps on exactly one struct field using structFieldNamer.
// In case of single column rows, nil must be passed for structFieldNamer.
// ScanRowsAsSlice calls srcRows.Close().
func ScanRowsAsSlice(ctx context.Context, srcRows Rows, dest any, structFieldNamer sqldb.StructFieldMapper) error {
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
			err := ScanStruct(srcRows, target.Interface(), structFieldNamer)
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

type SliceScanner struct {
	destSlice reflect.Value
}

func WrapWithSliceScanner(destPtr any) any {
	v := reflect.ValueOf(destPtr)
	if v.Elem().Kind() != reflect.Slice || v.Type().Implements(typeOfSQLScanner) {
		return destPtr
	}
	return SliceScanner{destSlice: v.Elem()}
}

// Scan implements the sql.Scanner interface.
func (a *SliceScanner) Scan(src any) error {
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
	elems, err := nullable.SplitArray(src)
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

func ScanValue(src any, dest reflect.Value) error {
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
