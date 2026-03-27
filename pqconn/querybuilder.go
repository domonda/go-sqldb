package pqconn

import "github.com/domonda/go-sqldb/postgres"

// QueryBuilder is an alias for [postgres.QueryBuilder] which implements
// [sqldb.QueryBuilder], [sqldb.UpsertQueryBuilder], and [sqldb.ReturningQueryBuilder]
// using PostgreSQL-compatible ON CONFLICT syntax.
type QueryBuilder = postgres.QueryBuilder
