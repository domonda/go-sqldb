package pqconn

import (
	"fmt"
	"regexp"
)

var columnNameRegex = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]{0,58}$`)

func validateColumnName(name string) error {
	if !columnNameRegex.MatchString(name) {
		return fmt.Errorf("invalid Postgres column name: %q", name)
	}
	return nil
}
