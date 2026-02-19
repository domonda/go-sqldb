package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// UpsertStruct upserts a row to table using the connection's
// StructFieldMapper to map struct fields to column names.
// If inserting conflicts on the primary key column(s), then an update is performed.
// Optional ColumnFilter can be passed to ignore mapped columns.
func UpsertStruct(ctx context.Context, table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return Conn(ctx).UpsertStruct(table, rowStruct, ignoreColumns...)
}
