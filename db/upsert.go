package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// UpsertStruct TODO
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	return sqldb.UpsertStruct(ctx, Conn(ctx), rowStruct, options...)
}

func UpsertStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, done func() error, err error) {
	return sqldb.UpsertStructStmt[S](ctx, Conn(ctx), options...)
}

func UpsertStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	return sqldb.UpsertStructs(ctx, Conn(ctx), rowStructs, options...)
}
