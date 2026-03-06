package sqliteconn

import "github.com/domonda/go-sqldb"

// QueryFormatter is the standard [sqldb.QueryFormatter] implementation
// used for SQLite (using `?` placeholders).
type QueryFormatter = sqldb.StdQueryFormatter

// QueryBuilder is the standard [sqldb.QueryBuilder] implementation
// used for SQLite.
type QueryBuilder = sqldb.StdQueryBuilder
