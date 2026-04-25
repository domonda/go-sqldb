package sqliteconn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	tableNameRegexp  = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]{0,127}\.)?[a-zA-Z_][a-zA-Z0-9_]{0,127}$`)
	columnNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,127}$`)
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
// The name must match the regex
// `^([a-zA-Z_][a-zA-Z0-9_]{0,127}\.)?[a-zA-Z_][a-zA-Z0-9_]{0,127}$`,
// optionally with a schema prefix separated by a dot. The name is then
// escaped with double-quotes using [EscapeIdentifier].
//
// SECURITY: this regex blocks identifier injection at the API surface.
// Even though [EscapeIdentifier] would handle embedded double-quotes
// correctly via doubling, validating the input shape gives defense in
// depth against accidental misuse and matches the behaviour of the other
// driver formatters in this module.
func (QueryFormatter) FormatTableName(name string) (string, error) {
	if !tableNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	if schema, table, ok := strings.Cut(name, "."); ok {
		return EscapeIdentifier(schema) + "." + EscapeIdentifier(table), nil
	}
	return EscapeIdentifier(name), nil
}

// FormatColumnName implements [sqldb.QueryFormatter.FormatColumnName].
// The name must match the regex `^[a-zA-Z_][a-zA-Z0-9_]{0,127}$` and is
// then escaped with double-quotes using [EscapeIdentifier].
//
// SECURITY: this regex blocks identifier injection at the API surface.
// See [QueryFormatter.FormatTableName] for the rationale.
func (QueryFormatter) FormatColumnName(name string) (string, error) {
	if !columnNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	return EscapeIdentifier(name), nil
}

// FormatPlaceholder implements [sqldb.QueryFormatter.FormatPlaceholder].
func (QueryFormatter) FormatPlaceholder(paramIndex int) string {
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
