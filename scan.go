package sqldb

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
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
// If the bytes are valid UTF-8 they are returned as a string unchanged.
// Otherwise they are encoded as uppercase hex (via fmt's %X verb) and
// prefixed with hexPrefix, producing strings like "\xDEADBEEF" or
// "0xDEADBEEF" depending on the prefix.
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
		return fmt.Sprintf("%s%X", hexPrefix, b), true
	}
}

// TimeToStringScanConverter returns a [ScanConverterFunc] that formats
// [driver.Value] of type [time.Time] as a string using the given layout
// (see [time.Time.Format]).
// Values that are not of type [time.Time] are returned as (nil, false)
// so the converter can be chained with others.
//
// Common layouts are [time.RFC3339], [time.RFC3339Nano],
// or [time.DateTime].
func TimeToStringScanConverter(layout string) ScanConverterFunc {
	return func(value driver.Value) (any, bool) {
		t, ok := value.(time.Time)
		if !ok {
			return nil, false
		}
		return t.Format(layout), true
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

// ScanDriverValue scans the [driver.Value] value into destPtr,
// converting it to the destination type on a best-effort basis.
//
// destPtr must be a non-nil pointer. The conversions mirror those of the
// standard library database/sql package as closely as possible:
//
//   - If destPtr implements [sql.Scanner] its Scan method is used.
//   - If destPtr implements decimalCompose and value implements
//     decimalDecompose the decimal value is composed directly.
//   - Pointers are followed: a nil value sets the pointer to nil, otherwise
//     a new value is allocated and scanned through.
//   - int64 and float64 values are converted between integer, unsigned and
//     floating point destinations with overflow and loss-of-precision checks.
//   - int64 and float64 values of 0 or 1 convert to bool.
//   - bool, int64, float64 and time.Time values can be formatted into string
//     and []byte destinations; string and []byte values can be parsed into
//     bool, integer, unsigned and floating point destinations.
//   - time.Time is formatted with [time.RFC3339Nano] for string and []byte
//     destinations.
//   - []byte values are cloned when assigned to a []byte or any destination
//     because the driver may reuse the backing array after the call.
//   - A nil value calls SetNull if destPtr implements interface{ SetNull() },
//     otherwise it sets slice and map destinations to nil.
func ScanDriverValue(destPtr any, value driver.Value) error {
	if destPtr == nil {
		return errNilDestPtr
	}

	dest := reflect.ValueOf(destPtr)
	// Reject a nil pointer destination up front so the [sql.Scanner] and
	// decimal paths below are not invoked on a nil pointer receiver.
	if dest.Kind() == reflect.Pointer && dest.IsNil() {
		return fmt.Errorf("%s %w", dest.Type(), errNilDestPtr)
	}

	if destScanner, ok := destPtr.(sql.Scanner); ok {
		return destScanner.Scan(value)
	}

	// Compose a decimal directly if both the destination and the value
	// support the database/sql decimal interfaces.
	if dst, ok := destPtr.(decimalCompose); ok {
		if src, ok := value.(decimalDecompose); ok {
			return dst.Compose(src.Decompose(nil))
		}
	}

	if dest.Kind() != reflect.Pointer {
		return fmt.Errorf("unable to scan non-pointer %s", dest.Type())
	}
	dest = dest.Elem()

	// destPtr is a pointer to an interface type
	if dest.Kind() == reflect.Interface {
		if value == nil {
			dest.SetZero() // Set nil
			return nil
		}
		// Clone driver bytes because the driver may reuse the backing array.
		if b, ok := value.([]byte); ok {
			value = bytes.Clone(b)
		}
		src := reflect.ValueOf(value)
		if !src.Type().AssignableTo(dest.Type()) {
			return fmt.Errorf("unable to scan %T into %s", value, dest.Type())
		}
		dest.Set(src)
		return nil
	}

	// destPtr is a pointer to a pointer type:
	// set nil for NULL, else allocate and scan through it
	if dest.Kind() == reflect.Pointer {
		if value == nil {
			dest.SetZero()
			return nil
		}
		if dest.IsNil() {
			dest.Set(reflect.New(dest.Type().Elem()))
		}
		return ScanDriverValue(dest.Interface(), value)
	}

	switch src := value.(type) {
	case int64:
		switch dest.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if dest.OverflowInt(src) {
				return fmt.Errorf("int64 value %d overflows %s", src, dest.Type())
			}
			dest.SetInt(src)
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if src < 0 {
				return fmt.Errorf("unable to scan negative int64 value %d into %s", src, dest.Type())
			}
			if dest.OverflowUint(uint64(src)) {
				return fmt.Errorf("int64 value %d overflows %s", src, dest.Type())
			}
			dest.SetUint(uint64(src))
			return nil
		case reflect.Float32, reflect.Float64:
			dest.SetFloat(float64(src))
			return nil
		case reflect.Bool:
			switch src {
			case 0:
				dest.SetBool(false)
				return nil
			case 1:
				dest.SetBool(true)
				return nil
			}
			return fmt.Errorf("unable to scan int64 value %d into %s", src, dest.Type())
		case reflect.String:
			dest.SetString(strconv.FormatInt(src, 10))
			return nil
		case reflect.Slice:
			if dest.Type().Elem().Kind() == reflect.Uint8 {
				dest.SetBytes(strconv.AppendInt(nil, src, 10))
				return nil
			}
		}

	case float64:
		switch dest.Kind() {
		case reflect.Float32, reflect.Float64:
			if dest.OverflowFloat(src) {
				return fmt.Errorf("float64 value %g overflows %s", src, dest.Type())
			}
			dest.SetFloat(src)
			return nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i := int64(src)
			if float64(i) != src {
				return fmt.Errorf("float64 value %g cannot be scanned into %s without loss", src, dest.Type())
			}
			if dest.OverflowInt(i) {
				return fmt.Errorf("float64 value %g overflows %s", src, dest.Type())
			}
			dest.SetInt(i)
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if src < 0 {
				return fmt.Errorf("unable to scan negative float64 value %g into %s", src, dest.Type())
			}
			u := uint64(src)
			if float64(u) != src {
				return fmt.Errorf("float64 value %g cannot be scanned into %s without loss", src, dest.Type())
			}
			if dest.OverflowUint(u) {
				return fmt.Errorf("float64 value %g overflows %s", src, dest.Type())
			}
			dest.SetUint(u)
			return nil
		case reflect.Bool:
			switch src {
			case 0:
				dest.SetBool(false)
				return nil
			case 1:
				dest.SetBool(true)
				return nil
			}
			return fmt.Errorf("unable to scan float64 value %g into %s", src, dest.Type())
		case reflect.String:
			dest.SetString(strconv.FormatFloat(src, 'g', -1, 64))
			return nil
		case reflect.Slice:
			if dest.Type().Elem().Kind() == reflect.Uint8 {
				dest.SetBytes(strconv.AppendFloat(nil, src, 'g', -1, 64))
				return nil
			}
		}

	case bool:
		switch dest.Kind() {
		case reflect.Bool:
			dest.SetBool(src)
			return nil
		case reflect.String:
			dest.SetString(strconv.FormatBool(src))
			return nil
		case reflect.Slice:
			if dest.Type().Elem().Kind() == reflect.Uint8 {
				dest.SetBytes(strconv.AppendBool(nil, src))
				return nil
			}
		}

	case []byte:
		switch dest.Kind() {
		case reflect.String:
			dest.SetString(string(src))
			return nil
		case reflect.Slice:
			if dest.Type().Elem().Kind() == reflect.Uint8 {
				// Clone because src may be reused by the driver after the call.
				dest.SetBytes(bytes.Clone(src))
				return nil
			}
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return scanStringInto(dest, string(src))
		}

	case string:
		switch dest.Kind() {
		case reflect.String:
			dest.SetString(src)
			return nil
		case reflect.Slice:
			if dest.Type().Elem().Kind() == reflect.Uint8 {
				dest.SetBytes([]byte(src))
				return nil
			}
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return scanStringInto(dest, src)
		}

	case time.Time:
		st := reflect.ValueOf(src)
		switch {
		case st.Type().AssignableTo(dest.Type()):
			dest.Set(st)
			return nil
		case dest.Kind() == reflect.String:
			dest.SetString(src.Format(time.RFC3339Nano))
			return nil
		case dest.Kind() == reflect.Slice && dest.Type().Elem().Kind() == reflect.Uint8:
			dest.SetBytes([]byte(src.Format(time.RFC3339Nano)))
			return nil
		case dest.Kind() == reflect.Struct && st.Type().ConvertibleTo(dest.Type()):
			// Named struct types with the same underlying type as time.Time,
			// e.g. "type MyTime time.Time".
			dest.Set(st.Convert(dest.Type()))
			return nil
		}

	case nil:
		if d, ok := destPtr.(interface{ SetNull() }); ok {
			d.SetNull()
			return nil
		}
		switch dest.Kind() {
		case reflect.Slice, reflect.Map:
			dest.SetZero()
			return nil
		}
	}

	return fmt.Errorf("unable to scan %#v as %T", value, destPtr)
}

// scanStringInto parses s into the bool, integer, unsigned or floating point
// destination dest, using the destination's bit size. It mirrors the
// string-intermediate numeric conversion of the standard library's
// database/sql package and also enables scanning into user defined types
// such as "type Int int64".
func scanStringInto(dest reflect.Value, s string) error {
	switch dest.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("unable to scan %q as %s: %w", s, dest.Type(), strconvErr(err))
		}
		dest.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, dest.Type().Bits())
		if err != nil {
			return fmt.Errorf("unable to scan %q as %s: %w", s, dest.Type(), strconvErr(err))
		}
		dest.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, dest.Type().Bits())
		if err != nil {
			return fmt.Errorf("unable to scan %q as %s: %w", s, dest.Type(), strconvErr(err))
		}
		dest.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, dest.Type().Bits())
		if err != nil {
			return fmt.Errorf("unable to scan %q as %s: %w", s, dest.Type(), strconvErr(err))
		}
		dest.SetFloat(f)
		return nil
	}
	return fmt.Errorf("unable to scan %q as %s", s, dest.Type())
}
