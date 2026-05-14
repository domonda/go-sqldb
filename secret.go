package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
)

var (
	_ Secret         = secret{}
	_ driver.Valuer  = secret{}
	_ sql.Scanner    = secret{}
	_ fmt.Stringer   = secret{}
	_ fmt.GoStringer = secret{}
)

// Secret wraps a query argument value to keep it from being revealed
// when a query is formatted for logging or error messages.
//
// A Secret still passes its wrapped value through to the database driver:
// Value (driver.Valuer) returns the wrapped value and Scan (sql.Scanner)
// scans into it. But [FormatValue] and [SubstitutePlaceholders] use the
// redacted String representation instead of the actual value, so secrets
// never appear in formatted queries, logs, or error call stacks.
type Secret interface {
	driver.Valuer
	sql.Scanner

	// Secret returns the wrapped secret value.
	Secret() any

	// String returns a redacted string that indicates the value
	// is a secret without revealing the actual value.
	String() string
}

// KeepSecret wraps the passed value in a [Secret] to prevent it from
// being revealed in formatted queries, logs, or error messages.
// The wrapped value is still passed through to the database driver.
func KeepSecret(val any) Secret {
	return secret{val}
}

type secret struct{ val any }

// Secret returns the wrapped secret value.
func (s secret) Secret() any {
	return s.val
}

// String returns a redacted placeholder instead of the secret value.
func (secret) String() string {
	return "***REDACTED***"
}

// GoString returns a redacted Go representation of the wrapped value.
func (s secret) GoString() string {
	return fmt.Sprintf("%T(***REDACTED***)", s.val)
}

// Value implements the database/sql/driver.Valuer interface
// by passing through the wrapped value.
func (s secret) Value() (driver.Value, error) {
	if valuer, ok := s.val.(driver.Valuer); ok {
		return valuer.Value()
	}
	return driver.DefaultParameterConverter.ConvertValue(s.val)
}

// Scan implements the database/sql.Scanner interface
// by scanning src into the wrapped value.
func (s secret) Scan(src any) error {
	return ScanDriverValue(s.val, src)
}
