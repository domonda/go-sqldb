package sqliteconn

import "github.com/domonda/go-sqldb"

// QueryFormatter is the [sqldb.QueryFormatter] implementation
// used for SQLite (using `?` placeholders).
type QueryFormatter struct {
	sqldb.StdQueryFormatter
}

func (QueryFormatter) MaxArgs() int {
	return 32766
}
