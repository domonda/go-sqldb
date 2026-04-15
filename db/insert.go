package db

import (
	"context"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values Values) error {
	conn := Conn(ctx)
	return sqldb.Insert(
		ctx,
		conn,
		QueryBuilder(ctx),
		conn,
		table,
		values,
	)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
// The configured [QueryBuilder] must implement [sqldb.UpsertQueryBuilder].
func InsertUnique(ctx context.Context, table string, values Values, onConflict string) (inserted bool, err error) {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.UpsertQueryBuilder)
	if !ok {
		return false, fmt.Errorf("db.InsertUnique: QueryBuilder %T does not implement sqldb.UpsertQueryBuilder", QueryBuilder(ctx))
	}
	return sqldb.InsertUnique(
		ctx,
		conn,
		builder,
		conn,
		table,
		values,
		onConflict,
	)
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
// The configured [QueryBuilder] must implement [sqldb.ReturningQueryBuilder].
func InsertReturning(ctx context.Context, table string, values Values, returning string) *sqldb.Row {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.ReturningQueryBuilder)
	if !ok {
		return sqldb.NewRow(
			sqldb.NewErrRows(fmt.Errorf("db.InsertReturning: QueryBuilder %T does not implement sqldb.ReturningQueryBuilder", QueryBuilder(ctx))),
			StructReflector(ctx),
			conn, // formatter
			"",   // query
			nil,  // args
		)
	}
	return sqldb.InsertReturning(
		ctx,
		conn,
		StructReflector(ctx),
		builder,
		conn,
		table,
		values,
		returning,
	)
}

// InsertRowStruct inserts a new row into the table for the given struct.
// Table name and column names are determined by the [StructReflector] from the context.
// The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"column"`).
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...QueryOption) error {
	conn := Conn(ctx)
	return sqldb.InsertRowStruct(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStruct,
		options...,
	)
}

// InsertRowStructStmt prepares a statement for inserting rows of type S.
// Table name and column names are determined by the [StructReflector] from the context.
// The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"column"`).
// Returns an insertFunc to insert individual rows and a closeStmt
// function that must be called when done.
func InsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.InsertRowStructStmt[S](
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		options...,
	)
}

// InsertUniqueRowStruct inserts a new row or does nothing if the onConflict statement applies.
// Returns true if a row was inserted.
// Table name and column names are determined by the [StructReflector] from the context.
// The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"column"`).
// Optional QueryOption can be passed to ignore mapped columns.
// The configured [QueryBuilder] must implement [sqldb.UpsertQueryBuilder].
func InsertUniqueRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, onConflict string, options ...QueryOption) (inserted bool, err error) {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.UpsertQueryBuilder)
	if !ok {
		return false, fmt.Errorf("db.InsertUniqueRowStruct: QueryBuilder %T does not implement sqldb.UpsertQueryBuilder", QueryBuilder(ctx))
	}
	return sqldb.InsertUniqueRowStruct(
		ctx,
		conn,
		StructReflector(ctx),
		builder,
		conn,
		rowStruct,
		onConflict,
		options...,
	)
}

// InsertRowStructs inserts a slice of structs as new rows into the table for the given struct type.
// Rows are batched into multi-row INSERT statements respecting the driver's MaxArgs() limit.
//
// Optimization strategy:
//   - Single row: delegates to [InsertRowStruct] (benefits from the query cache).
//   - Single batch (all rows fit within MaxArgs): executes a single multi-row INSERT directly
//     without a transaction or prepared statement.
//   - Multiple batches: wraps all batches in a transaction for atomicity.
//     When there are 2+ full batches, a prepared statement is created and reused
//     across all full batches to avoid repeated query parsing on the server.
//     Any remainder rows are executed as a separate, smaller multi-row INSERT.
//
// Table name and column names are determined by the [StructReflector] from the context.
// The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"column"`).
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...QueryOption) error {
	conn := Conn(ctx)
	return sqldb.InsertRowStructs(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStructs,
		options...,
	)
}
