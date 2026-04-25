package db

import (
	"context"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// Update table row(s) with values using the whereCondition with passed in
// args bound positionally to whereCondition placeholders.
//
// whereCondition is the boolean expression that follows the WHERE keyword
// and must NOT include the WHERE keyword itself. Use the driver's placeholder
// syntax for parameters bound to args (see [sqldb] package documentation).
// Examples using PostgreSQL/SQLite syntax: "id = $1",
// "tenant_id = $1 AND status IN ($2, $3)".
//
// SECURITY: whereCondition is concatenated into the generated SQL verbatim
// and is NOT parameterized. It must be static SQL written by the developer.
// Pass external input only through args using the driver's placeholder syntax.
func Update(ctx context.Context, table string, values Values, whereCondition string, args ...any) error {
	conn := Conn(ctx)
	return sqldb.Update(
		ctx,
		conn,
		QueryBuilder(ctx),
		conn,
		table,
		values,
		whereCondition,
		args...,
	)
}

// UpdateReturningRow updates a table row with values using the whereCondition
// with passed in args bound positionally to whereCondition placeholders and
// returns a Row for scanning the columns specified in returningColumns.
// The configured [QueryBuilder] must implement [sqldb.ReturningQueryBuilder].
//
// whereCondition follows the WHERE keyword without including it;
// returningColumns follows the RETURNING keyword without including it.
//
// SECURITY: both returningColumns and whereCondition are concatenated into
// the SQL verbatim and are NOT parameterized. Pass external input only
// through args using the driver's placeholder syntax.
func UpdateReturningRow(ctx context.Context, table string, values Values, returningColumns, whereCondition string, args ...any) *sqldb.Row {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.ReturningQueryBuilder)
	if !ok {
		return sqldb.NewRow(
			sqldb.NewErrRows(fmt.Errorf("db.UpdateReturningRow: QueryBuilder %T does not implement sqldb.ReturningQueryBuilder", QueryBuilder(ctx))),
			StructReflector(ctx),
			conn, // formatter
			"",   // query
			nil,  // args
		)
	}
	return sqldb.UpdateReturningRow(
		ctx,
		conn,
		StructReflector(ctx),
		builder,
		conn,
		table,
		values,
		returningColumns,
		whereCondition,
		args...,
	)
}

// UpdateReturningRows updates table rows with values using the whereCondition
// with passed in args bound positionally to whereCondition placeholders and
// returns Rows for scanning the columns specified in returningColumns.
// The configured [QueryBuilder] must implement [sqldb.ReturningQueryBuilder].
//
// whereCondition follows the WHERE keyword without including it;
// returningColumns follows the RETURNING keyword without including it.
//
// SECURITY: both returningColumns and whereCondition are concatenated into
// the SQL verbatim and are NOT parameterized. Pass external input only
// through args using the driver's placeholder syntax.
func UpdateReturningRows(ctx context.Context, table string, values Values, returningColumns, whereCondition string, args ...any) sqldb.Rows {
	conn := Conn(ctx)
	builder, ok := QueryBuilder(ctx).(sqldb.ReturningQueryBuilder)
	if !ok {
		return sqldb.NewErrRows(fmt.Errorf("db.UpdateReturningRows: QueryBuilder %T does not implement sqldb.ReturningQueryBuilder", QueryBuilder(ctx)))
	}
	return sqldb.UpdateReturningRows(
		ctx,
		conn,
		builder,
		conn,
		table,
		values,
		returningColumns,
		whereCondition,
		args...,
	)
}

// UpdateRowStruct updates a row in a table using the exported fields of rowStruct.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"column"`, `db:"id,primarykey"`).
// Struct fields can be filtered with options like [IgnoreColumns] or [OnlyColumns].
// The struct must have at least one primary key field.
func UpdateRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...QueryOption) error {
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
// (e.g., db.TableName `db:"my_table"`, field `db:"column"`, `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns an updateFunc to update individual rows and a closeStmt
// function that must be called when done.
func UpdateRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...QueryOption) (updateFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
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
// (e.g., db.TableName `db:"my_table"`, field `db:"column"`, `db:"id,primarykey"`).
// The struct must have at least one primary key field.
func UpdateRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...QueryOption) error {
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
