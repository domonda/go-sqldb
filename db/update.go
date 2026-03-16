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

// UpdateRowStruct updates a row in a table using the exported fields of rowStruct.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"column"`, `db:"id,primarykey"`).
// Struct fields can be filtered with options like [sqldb.IgnoreColumns] or [sqldb.OnlyColumns].
// The struct must have at least one primary key field.
func UpdateRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	return sqldb.UpdateRowStruct(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStruct,
		options...,
	)
}

// UpdateRowStructStmt prepares a statement for updating rows of type S.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"column"`, `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns an updateFunc to update individual rows and a closeStmt
// function that must be called when done.
func UpdateRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (updateFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.UpdateRowStructStmt[S](
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		options...,
	)
}

// UpdateRowStructs updates a slice of structs within a transaction
// using a prepared statement for efficiency.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"column"`, `db:"id,primarykey"`).
// The struct must have at least one primary key field.
func UpdateRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	return sqldb.UpdateRowStructs(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStructs,
		options...,
	)
}
