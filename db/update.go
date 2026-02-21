package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, table string, values sqldb.Values, where string, args ...any) error {
	return sqldb.Update(ctx, Conn(ctx), table, values, where, args...)
}

// UpdateRowStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
func UpdateRowStruct(ctx context.Context, table string, rowStruct any, options ...sqldb.QueryOption) error {
	return sqldb.UpdateRowStruct(ctx, Conn(ctx), table, rowStruct, options...)
}
