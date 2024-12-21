package mockconn

import (
	"errors"
	"fmt"
	"regexp"
)

var ErrMockedScan = errors.New("mocked scan")

var columnNameRegex = regexp.MustCompile(`^[0-9a-zA-Z_]{1,64}$`)

func validateColumnName(name string) error {
	if !columnNameRegex.MatchString(name) {
		return fmt.Errorf("invalid MySQL column name: %q", name)
	}
	return nil
}
