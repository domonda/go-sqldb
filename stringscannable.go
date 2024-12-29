package sqldb

import "fmt"

// StringScannable implements the sql.Scanner interface
// and converts all scanned values to string.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint(src).
type StringScannable string

// Scan implements implements the sql.Scanner interface
// and converts all scanned values to string.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint(src).
func (s *StringScannable) Scan(src any) error {
	switch src := src.(type) {
	case nil:
		*s = ""
	case string:
		*s = StringScannable(src)
	case []byte:
		*s = StringScannable(src)
	default:
		*s = StringScannable(fmt.Sprint(src))
	}
	return nil
}
