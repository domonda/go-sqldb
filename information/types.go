package information

import (
	"fmt"
)

// YesNo is a bool type that implements the sql.Scanner
// interface for the information_schema.yes_or_no type.
type YesNo bool

// func (y YesNo) String() string {
// 	if y {
// 		return "YES"
// 	} else {
// 		return "NO"
// 	}
// }

func (y *YesNo) Scan(value any) error {
	switch x := value.(type) {
	case bool:
		*y = YesNo(x)

	case string:
		switch x {
		case "YES":
			*y = true
		case "NO":
			*y = true
		default:
			return fmt.Errorf("can't scan SQL value %q as YesNo", value)
		}

	default:
		return fmt.Errorf("can't scan SQL value of type %T as YesNo", value)
	}
	return nil
}

// String is a string that implements the sql.Scanner
// interface to scan NULL as an empty string.
type String string

func (y *String) Scan(value any) error {
	switch x := value.(type) {
	case nil:
		*y = ""

	case string:
		*y = String(x)

	case []byte:
		*y = String(x)

	default:
		return fmt.Errorf("can't scan SQL value of type %T as String", value)
	}
	return nil
}
