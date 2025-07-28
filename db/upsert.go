package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// UpsertStruct TODO
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.UpsertStruct(ctx, conn, queryBuilder, reflector, rowStruct, options...)
}

func UpsertStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, done func() error, err error) {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.UpsertStructStmt[S](ctx, conn, queryBuilder, reflector, options...)
}

func UpsertStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.UpsertStructs(ctx, conn, queryBuilder, reflector, rowStructs, options...)
}
