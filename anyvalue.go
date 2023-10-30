package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"slices"
	"unicode/utf8"
)

var (
	_ sql.Scanner    = &AnyValue{}
	_ driver.Valuer  = AnyValue{}
	_ fmt.Stringer   = AnyValue{}
	_ fmt.GoStringer = AnyValue{}
)

// AnyValue can hold any value and is useful for
// generic code that can handle unknown column types.
//
// AnyValue implements the following interfaces:
//   - database/sql.Scanner
//   - database/sql/driver.Valuer
//   - fmt.Stringer
//   - fmt.GoStringer
//
// When scanned with the Scan method
// Val will have one of the following types:
//   - int64
//   - float64
//   - bool
//   - []byte
//   - string
//   - time.Time
//   - nil (for SQL NULL values)
type AnyValue struct {
	Val any
}

// Scan implements the database/sql.Scanner interface.
func (any *AnyValue) Scan(val any) error {
	if b, ok := val.([]byte); ok {
		// Copy bytes because they won't be valid after this method call
		any.Val = slices.Clone(b)
	} else {
		any.Val = val
	}
	return nil
}

// Value implements the driver database/sql/driver.Valuer interface.
func (any AnyValue) Value() (driver.Value, error) {
	return any.Val, nil
}

// String returns the value formatted as string using fmt.Sprint
// except when it is of type []byte and valid UTF-8,
// then it is directly converted into a string.
func (any AnyValue) String() string {
	if b, ok := any.Val.([]byte); ok && utf8.Valid(b) {
		return string(b)
	}
	return fmt.Sprint(any.Val)
}

// GoString returns a Go representation of the wrapped value.
func (any AnyValue) GoString() string {
	if b, ok := any.Val.([]byte); ok && utf8.Valid(b) {
		return fmt.Sprintf("[]byte(%#v)", string(b))
	}
	return fmt.Sprintf("%#v", any.Val)
}
