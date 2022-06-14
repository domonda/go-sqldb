package mysqlconn

import (
	"fmt"
	"regexp"
)

var columnNameRegex = regexp.MustCompile(`^[0-9a-zA-Z$_]{1,64}$`)

func validateColumnName(name string) error {
	if !columnNameRegex.MatchString(name) {
		return fmt.Errorf("invalid MySQL column name: %q", name)
	}
	return nil
}
