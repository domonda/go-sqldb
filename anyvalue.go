package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"unicode/utf8"
)

var (
	_ sql.Scanner    = &AnyValue{}
	_ driver.Valuer  = AnyValue{}
	_ fmt.Stringer   = AnyValue{}
	_ fmt.GoStringer = AnyValue{}
)

// AnyValue wraps a driver.Value and is useful for
// generic code that can handle unknown column types.
//
// AnyValue implements the following interfaces:
//   database/sql.Scanner
//   database/sql/driver.Valuer
//   fmt.Stringer
//   fmt.GoStringer
//
// When scanned, Val can have one of the following underlying types:
//   int64
//   float64
//   bool
//   []byte
//   string
//   time.Time
//   nil - for NULL values
type AnyValue struct {
	Val any
}

// Scan implements the database/sql.Scanner interface.
func (a *AnyValue) Scan(val any) error {
	if b, ok := val.([]byte); ok {
		// Copy bytes because they won't be valid after this method call
		a.Val = append([]byte(nil), b...)
	} else {
		a.Val = val
	}
	return nil
}

// Value implements the driver database/sql/driver.Valuer interface.
func (a AnyValue) Value() (driver.Value, error) {
	return a.Val, nil
}

// String returns the value formatted as string using fmt.Sprint
// except when it's of type []byte and valid UTF-8,
// then it is directly converted into a string.
func (a AnyValue) String() string {
	if b, ok := a.Val.([]byte); ok && utf8.Valid(b) {
		return string(b)
	}
	return fmt.Sprint(a.Val)
}

// GoString returns a Go representation of the wrapped value.
func (a AnyValue) GoString() string {
	if b, ok := a.Val.([]byte); ok && utf8.Valid(b) {
		return fmt.Sprintf("[]byte(%q)", b)
	}
	return fmt.Sprintf("%#v", a.Val)
}
