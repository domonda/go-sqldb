package pqconn

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	tableNameRegexp = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]{0,62}\.)?[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)
	columnNameRegex = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]{0,58}$`)
)

type QueryFormatter struct{}

func (QueryFormatter) FormatTableName(name string) (string, error) {
	if !tableNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	return name, nil
}

func (QueryFormatter) FormatColumnName(name string) (string, error) {
	if !columnNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	return name, nil
}

func (f QueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	return "$" + strconv.Itoa(paramIndex+1)
}
