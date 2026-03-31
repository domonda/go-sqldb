package db

import (
	"context"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// UpsertRowStruct inserts a new row or updates an existing one
// if inserting conflicts on the primary key column(s).
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// The configured [QueryBuilder] must implement [sqldb.UpsertQueryBuilder].
func UpsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.UpsertQueryBuilder)
	if !ok {
		return fmt.Errorf("db.UpsertRowStruct: QueryBuilder %T does not implement sqldb.UpsertQueryBuilder", QueryBuilder(ctx))
	}
	return sqldb.UpsertRowStruct(
		ctx,
		conn,
		StructReflector(ctx),
		builder,
		conn,
		rowStruct,
		options...,
	)
}

// UpsertRowStructStmt prepares a statement for upserting rows of type S.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns an upsert function and a closeStmt function that must be called when done.
// The configured [QueryBuilder] must implement [sqldb.UpsertQueryBuilder].
func UpsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.UpsertQueryBuilder)
	if !ok {
		return nil, nil, fmt.Errorf("db.UpsertRowStructStmt: QueryBuilder %T does not implement sqldb.UpsertQueryBuilder", QueryBuilder(ctx))
	}
	return sqldb.UpsertRowStructStmt[S](
		ctx,
		conn,
		StructReflector(ctx),
		builder,
		conn,
		options...,
	)
}

// UpsertRowStructs upserts a slice of structs within a transaction
// using a prepared statement.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The configured [QueryBuilder] must implement [sqldb.UpsertQueryBuilder].
func UpsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.UpsertQueryBuilder)
	if !ok {
		return fmt.Errorf("db.UpsertRowStructs: QueryBuilder %T does not implement sqldb.UpsertQueryBuilder", QueryBuilder(ctx))
	}
	return sqldb.UpsertRowStructs(
		ctx,
		conn,
		StructReflector(ctx),
		builder,
		conn,
		rowStructs,
		options...,
	)
}
