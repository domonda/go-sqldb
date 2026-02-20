package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// UpsertStruct inserts a new row or updates an existing one
// if inserting conflicts on the primary key column(s).
func UpsertStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	return sqldb.UpsertStruct(ctx, Conn(ctx), rowStruct, options...)
}

// UpsertStructStmt prepares a statement for upserting rows of type S.
// Returns an upsert function and a done function that must be called when done.
func UpsertStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, done func() error, err error) {
	return sqldb.UpsertStructStmt[S](ctx, Conn(ctx), options...)
}

// UpsertStructs upserts a slice of structs within a transaction
// using a prepared statement.
func UpsertStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	return sqldb.UpsertStructs(ctx, Conn(ctx), rowStructs, options...)
}
