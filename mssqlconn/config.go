package mssqlconn

import (
	"fmt"
	"regexp"
)

const (
	Driver = "sqlserver"

	argFmt = "@p%d"
)

// TODO allow spaces and other characters and escaping with backticks
var columnNameRegex = regexp.MustCompile(`^[0-9a-zA-Z$_]{1,64}$`)

// https://learn.microsoft.com/en-us/sql/odbc/microsoft/column-name-limitations
func validateColumnName(name string) error {
	if !columnNameRegex.MatchString(name) {
		return fmt.Errorf("invalid Microsoft SQL Server column name: %q", name)
	}
	return nil
}
