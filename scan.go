package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"
)

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
