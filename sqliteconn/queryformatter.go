package sqliteconn

import (
	"fmt"
	"strconv"
	"strings"
)

// EscapeIdentifier wraps a SQLite identifier in double-quotes,
// escaping any embedded double-quote characters as "".
func EscapeIdentifier(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// QueryFormatter is the [sqldb.QueryFormatter] implementation
// used for SQLite with `?NNN` positional placeholders (?1, ?2, ...).
// SQLite supports both sequential `?` and positional `?NNN` placeholders.
// Positional placeholders are required so that query builders like
// [sqldb.StdQueryBuilder.UpdateColumns] can reference arguments by index
// regardless of their order in the SQL statement.
type QueryFormatter struct{}

// FormatTableName implements [sqldb.QueryFormatter.FormatTableName].
// It validates that the name is not empty and escapes it
// with double-quotes using [EscapeIdentifier].
// An optional schema prefix separated by a dot is supported.
func (QueryFormatter) FormatTableName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty table name")
	}
	schema, table, hasSchema := strings.Cut(name, ".")
	if hasSchema {
		if schema == "" || table == "" {
			return "", fmt.Errorf("invalid table name %q", name)
		}
		return EscapeIdentifier(schema) + "." + EscapeIdentifier(table), nil
	}
	return EscapeIdentifier(name), nil
}

// FormatColumnName implements [sqldb.QueryFormatter.FormatColumnName].
// It validates that the name is not empty and escapes it
// with double-quotes using [EscapeIdentifier].
func (QueryFormatter) FormatColumnName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty column name")
	}
	return EscapeIdentifier(name), nil
}

// FormatPlaceholder implements [sqldb.QueryFormatter.FormatPlaceholder].
func (QueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	return "?" + strconv.Itoa(paramIndex+1)
}

// FormatStringLiteral implements [sqldb.QueryFormatter.FormatStringLiteral].
func (QueryFormatter) FormatStringLiteral(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "''") + "'"
}

// MaxArgs implements [sqldb.QueryFormatter.MaxArgs].
func (QueryFormatter) MaxArgs() int {
	return 32766
}
