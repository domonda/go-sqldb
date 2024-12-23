package mssqlconn

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/domonda/go-sqldb"
)

// TODO allow spaces and other characters and escaping with backticks
// https://learn.microsoft.com/en-us/sql/odbc/microsoft/column-name-limitations
var columnNameRegex = regexp.MustCompile(`^[0-9a-zA-Z$_]{1,64}$`)

var tableNameRegex = columnNameRegex

type QueryFormatter struct{}

func (QueryFormatter) FormatTableName(name string) (string, error) {
	if !tableNameRegex.MatchString(name) {
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
	return "@p" + strconv.Itoa(paramIndex+1)
}

func (QueryFormatter) FormatStringLiteral(str string) string {
	return sqldb.FormatSingleQuoteStringLiteral(str)
}
