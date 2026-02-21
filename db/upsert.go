package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// UpsertRowStruct inserts a new row or updates an existing one
// if inserting conflicts on the primary key column(s).
func UpsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	return sqldb.UpsertRowStruct(ctx, Conn(ctx), rowStruct, options...)
}

// UpsertRowStructStmt prepares a statement for upserting rows of type S.
// Returns an upsert function and a closeStmt function that must be called when done.
func UpsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	return sqldb.UpsertRowStructStmt[S](ctx, Conn(ctx), options...)
}

// UpsertRowStructs upserts a slice of structs within a transaction
// using a prepared statement.
func UpsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	return sqldb.UpsertRowStructs(ctx, Conn(ctx), rowStructs, options...)
}
