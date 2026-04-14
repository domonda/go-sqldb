package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"time"
	"unicode/utf8"
)

// ScanConverter converts a [driver.Value] into another value
// before it is scanned into a destination by [ScanDriverValue].
// The second return value reports whether the conversion was applied;
// if false the original value is used unchanged.
type ScanConverter interface {
	ConvertValue(value driver.Value) (any, bool)
}

// ScanConverterFunc adapts a function to the [ScanConverter] interface.
type ScanConverterFunc func(value driver.Value) (any, bool)

// ConvertValue calls f(value).
func (f ScanConverterFunc) ConvertValue(value driver.Value) (any, bool) {
	return f(value)
}

// ScanConverters is a slice of [ScanConverter] that itself implements
// [ScanConverter] by trying each converter in order and returning
// the result of the first one that reports a successful conversion.
type ScanConverters []ScanConverter

// ConvertValue returns the result of the first converter in converters
// that reports a successful conversion of value,
// or (nil, false) if no converter applied.
func (converters ScanConverters) ConvertValue(value driver.Value) (any, bool) {
	for _, converter := range converters {
		if val, ok := converter.ConvertValue(value); ok {
			return val, true
		}
	}
	return nil, false
}

// BytesToStringScanConverter returns a [ScanConverterFunc] that converts
// [driver.Value] of type []byte to a string.
// If the bytes are valid UTF-8 they are returned as a string,
// otherwise they are hex-encoded and prefixed with hexPrefix.
// Values that are not of type []byte are returned as (nil, false)
// so the converter can be chained with others.
//
// Common hexPrefix values are `\x` (PostgreSQL bytea hex format)
// or `0x` (Go and SQL hex literal).
func BytesToStringScanConverter(hexPrefix string) ScanConverterFunc {
	return func(value driver.Value) (any, bool) {
		b, ok := value.([]byte)
		if !ok {
			return nil, false
		}
		if utf8.Valid(b) {
			return string(b), true
		}
		return hexPrefix + hex.EncodeToString(b), true
	}
}

var (
	_ ScanConverter = ScanConverterFunc(nil)
	_ ScanConverter = ScanConverters(nil)
)

// ScanConvertValueOrUnchanged returns the result of the first converter
// that reports a successful conversion of value,
// or value unchanged if no converter applied.
func ScanConvertValueOrUnchanged(value any, converters ...ScanConverter) any {
	for _, converter := range converters {
		if val, ok := converter.ConvertValue(value); ok {
			return val
		}
	}
	return value
}

// ScanDriverValue scans a [driver.Value] into destPtr.
func ScanDriverValue(destPtr any, value driver.Value) error {
	if destPtr == nil {
		return errors.New("unable to scan nil destPtr")
	}

	if destScanner, ok := destPtr.(sql.Scanner); ok {
		return destScanner.Scan(value)
	}

	dest := reflect.ValueOf(destPtr)
	if dest.Kind() != reflect.Pointer {
		return fmt.Errorf("unable to scan non-pointer %s", dest.Type())
	}
	dest = dest.Elem()

	// destPtr is a pointer to any type
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
			if src < 0 {
				return fmt.Errorf("unable to scan negative int64 value %d into %s", src, dest.Type())
			}
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
			if src < 0 {
				return fmt.Errorf("unable to scan negative float64 value %f into %s", src, dest.Type())
			}
			dest.SetUint(uint64(src))
			return nil
		}

	case bool:
		if dest.Kind() == reflect.Bool {
			dest.SetBool(src)
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
		case reflect.Pointer, reflect.Slice, reflect.Map:
			dest.SetZero()
			return nil
		}
	}

	return fmt.Errorf("unable to scan %#v as %T", value, destPtr)
}
