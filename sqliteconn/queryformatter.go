package sqliteconn

import (
	"strconv"
	"strings"

	"github.com/domonda/go-sqldb"
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
type QueryFormatter struct {
	sqldb.StdQueryFormatter
}

func (QueryFormatter) FormatPlaceholder(paramIndex int) string {
	return "?" + strconv.Itoa(paramIndex+1)
}

// MaxArgs implements [sqldb.QueryFormatter.MaxArgs].
func (QueryFormatter) MaxArgs() int {
	return 32766
}
