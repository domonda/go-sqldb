package information

import (
	"fmt"
)

// YesNo is a bool type that implements the sql.Scanner
// interface for the information_schema.yes_or_no type.
type YesNo bool

// Scan implements the sql.Scanner interface for YesNo.
func (y *YesNo) Scan(value any) error {
	switch x := value.(type) {
	case bool:
		*y = YesNo(x)

	case string:
		switch x {
		case "YES":
			*y = true
		case "NO":
			*y = false
		default:
			return fmt.Errorf("unable to scan SQL value %q as YesNo", value)
		}

	default:
		return fmt.Errorf("unable to scan SQL value of type %T as YesNo", value)
	}
	return nil
}

// String is a string that implements the sql.Scanner
// interface to scan strings with SQL NULL as empty string.
type String string

// Scan implements the sql.Scanner interface for String.
func (y *String) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*y = ""
	case string:
		*y = String(value)
	case []byte:
		*y = String(value)
	default:
		return fmt.Errorf("unable to scan SQL value of type %T as String", value)
	}
	return nil
}
