package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// UpsertRowStruct inserts a new row or updates an existing one
// if inserting conflicts on the primary key column(s).
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The struct must have at least one primary key field.
func UpsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	return sqldb.UpsertRowStruct(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStruct,
		options...,
	)
}

// UpsertRowStructStmt prepares a statement for upserting rows of type S.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns an upsert function and a closeStmt function that must be called when done.
func UpsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.UpsertRowStructStmt[S](
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		options...,
	)
}

// UpsertRowStructs upserts a slice of structs within a transaction
// using a prepared statement.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"id,primarykey"`).
func UpsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	return sqldb.UpsertRowStructs(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStructs,
		options...,
	)
}
