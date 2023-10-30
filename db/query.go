package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// InsertStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStruct(ctx context.Context, table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return Conn(ctx).InsertStruct(table, rowStruct, ignoreColumns...)
}
