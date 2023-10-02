package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// ScanValues returns the values of a row exactly how they are
// passed from the database driver to an sql.Scanner.
// Byte slices will be copied.
func ScanValues(src Row) ([]any, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	var (
		anys   = make([]AnyValue, len(cols))
		result = make([]any, len(cols))
	)
	// result elements hold pointer to AnyValue for scanning
	for i := range result {
		result[i] = &anys[i]
	}
	err = src.Scan(result...)
	if err != nil {
		return nil, err
	}
	// don't return pointers to AnyValue
	// but what internal value has been scanned
	for i := range result {
		result[i] = anys[i].Val
	}
	return result, nil
}

// ScanStrings scans the values of a row as strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint.
func ScanStrings(src Row) ([]string, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	var (
		result     = make([]string, len(cols))
		resultPtrs = make([]any, len(cols))
	)
	for i := range resultPtrs {
		resultPtrs[i] = (*StringScannable)(&result[i])
	}
	err = src.Scan(resultPtrs...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ScanStruct(srcRow Row, destStruct any, mapper StructFieldMapper) error {
	v := reflect.ValueOf(destStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	var (
		setDestStructPtr = false
		destStructPtr    reflect.Value
		newStructPtr     reflect.Value
	)
	if v.Kind() == reflect.Ptr && v.IsNil() && v.CanSet() {
		// Got a nil pointer that we can set with a newly allocated struct
		setDestStructPtr = true
		destStructPtr = v
		newStructPtr = reflect.New(v.Type().Elem())
		// Continue with the newly allocated struct
		v = newStructPtr.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("ScanStruct: expected struct but got %T", destStruct)
	}

	columns, err := srcRow.Columns()
	if err != nil {
		return err
	}

	fieldPointers, err := MapStructFieldPointersForColumns(mapper, v, columns)
	if err != nil {
		return fmt.Errorf("ScanStruct: %w", err)
	}

	err = srcRow.Scan(fieldPointers...)
	if err != nil {
		return err
	}

	if setDestStructPtr {
		destStructPtr.Set(newStructPtr)
	}

	return nil
}

// ScanDriverValue scans a driver.Value into destPtr.
func ScanDriverValue(destPtr any, value driver.Value) error {
	if destPtr == nil {
		return errors.New("can't scan nil destPtr")
	}

	if destScanner, ok := destPtr.(sql.Scanner); ok {
		return destScanner.Scan(value)
	}

	dest := reflect.ValueOf(destPtr)
	if dest.Kind() != reflect.Ptr {
		return fmt.Errorf("can't scan non-pointer %s", dest.Type())
	}
	dest = dest.Elem()

	// destPtr is a pointer to interface{} type
	if dest.Kind() == reflect.Interface {
		if value != nil {
			dest.Set(reflect.ValueOf(value)) // Assign any
		} else {
			dest.SetZero() // Set nil
		}
		return nil
	}

	switch src := value.(type) {
	case int64:
		switch dest.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			dest.SetInt(src)
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dest.SetUint(uint64(src))
			return nil
		case reflect.Float32, reflect.Float64:
			dest.SetFloat(float64(src))
			return nil
		}

	case float64:
		switch dest.Kind() {
		case reflect.Float32, reflect.Float64:
			dest.SetFloat(src)
			return nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			dest.SetInt(int64(src))
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dest.SetUint(uint64(src))
			return nil
		}

	case bool:
		if dest.Kind() == reflect.Bool {
			reflect.ValueOf(destPtr).SetBool(src)
			return nil
		}

	case []byte:
		switch {
		case dest.Kind() == reflect.String:
			dest.SetString(string(src))
			return nil
		case dest.Kind() == reflect.Slice && dest.Type().Elem().Kind() == reflect.Uint8:
			dest.SetBytes(append([]byte(nil), src...)) // Make copy because src will be invalid after call
			return nil
		}

	case string:
		switch {
		case dest.Kind() == reflect.String:
			dest.SetString(src)
			return nil
		case dest.Kind() == reflect.Slice && dest.Type().Elem().Kind() == reflect.Uint8:
			dest.SetBytes([]byte(src))
			return nil
		}

	case time.Time:
		if s := reflect.ValueOf(value); s.Type().AssignableTo(dest.Type()) {
			dest.Set(s)
			return nil
		}

	case nil:
		if d, ok := destPtr.(interface{ SetNull() }); ok {
			d.SetNull()
			return nil
		}
		switch dest.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map:
			dest.SetZero()
			return nil
		}
	}

	return fmt.Errorf("can't scan %#v as %T", value, destPtr)
}
