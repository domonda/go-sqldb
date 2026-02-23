package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// UpsertRowStruct inserts a new row or updates an existing one
// if inserting conflicts on the primary key column(s).
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The struct must have at least one primary key field.
func UpsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	return sqldb.UpsertRowStruct(ctx, Conn(ctx), rowStruct, options...)
}

// UpsertRowStructStmt prepares a statement for upserting rows of type S.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns an upsert function and a closeStmt function that must be called when done.
func UpsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	return sqldb.UpsertRowStructStmt[S](ctx, Conn(ctx), options...)
}

// UpsertRowStructs upserts a slice of structs within a transaction
// using a prepared statement.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
func UpsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	return sqldb.UpsertRowStructs(ctx, Conn(ctx), rowStructs, options...)
}
