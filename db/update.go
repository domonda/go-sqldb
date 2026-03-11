package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Update table row(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, table string, values sqldb.Values, where string, args ...any) error {
	conn := Conn(ctx)
	return sqldb.Update(
		ctx,
		conn,
		QueryBuilder(ctx),
		conn,
		table,
		values,
		where,
		args...,
	)
}

// UpdateRowStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields can be filtered with options like [sqldb.IgnoreColumns] or [sqldb.OnlyColumns].
// The struct must have at least one field with a `db` tag value having a ",primarykey" suffix
// to mark primary key column(s).
func UpdateRowStruct(ctx context.Context, table string, rowStruct any, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	return sqldb.UpdateRowStruct(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		table,
		rowStruct,
		options...,
	)
}
