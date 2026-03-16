package sqliteconn

import (
	"strings"

	"github.com/domonda/go-sqldb"
)

// EscapeIdentifier wraps a SQLite identifier in double-quotes,
// escaping any embedded double-quote characters as "".
func EscapeIdentifier(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// QueryFormatter is the [sqldb.QueryFormatter] implementation
// used for SQLite (using `?` placeholders).
type QueryFormatter struct {
	sqldb.StdQueryFormatter
}

func (QueryFormatter) MaxArgs() int {
	return 32766
}
