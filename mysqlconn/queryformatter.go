package mysqlconn

import "github.com/domonda/go-sqldb"

// QueryFormatter is the standard [sqldb.QueryFormatter] implementation
// used for MySQL (using `?` placeholders).
type QueryFormatter = sqldb.StdQueryFormatter

// QueryBuilder is the standard [sqldb.QueryBuilder] implementation
// used for MySQL.
type QueryBuilder = sqldb.StdQueryBuilder
