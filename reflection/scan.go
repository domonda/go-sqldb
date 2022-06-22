package reflection

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"
)

func ScanValue(src driver.Value, dest reflect.Value) error {
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
