package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, table string, values sqldb.Values, where string, args ...any) error {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
	)
	return sqldb.Update(ctx, conn, queryBuilder, table, values, where, args...)
}

// UpdateStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
func UpdateStruct(ctx context.Context, table string, rowStruct any, options ...sqldb.QueryOption) error {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.UpdateStruct(ctx, conn, queryBuilder, reflector, table, rowStruct, options...)
}
