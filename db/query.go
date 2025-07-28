package db

import (
	"context"
	"time"

	"github.com/domonda/go-sqldb"
)

// CurrentTimestamp returns the SQL CURRENT_TIMESTAMP
// for the connection added to the context
// or else the default connection.
//
// Returns time.Now() in case of any error.
//
// Useful for getting the timestamp of a
// SQL transaction for use in Go code.
func CurrentTimestamp(ctx context.Context) time.Time {
	t, err := QueryRowValue[time.Time](ctx, `SELECT CURRENT_TIMESTAMP`)
	if err != nil {
		return time.Now()
	}
	return t
}

// QueryRow queries a single row and returns a Row for the results.
func QueryRow(ctx context.Context, query string, args ...any) *sqldb.Row {
	var (
		conn      = Conn(ctx)
		reflector = GetStructReflector(ctx)
	)
	return sqldb.QueryRow(ctx, conn, reflector, query, args...)
}

// QueryRowValue queries a single row mapped to the type T.
func QueryRowValue[T any](ctx context.Context, query string, args ...any) (val T, err error) {
	var (
		conn      = Conn(ctx)
		reflector = GetStructReflector(ctx)
	)
	return sqldb.QueryRowValue[T](ctx, conn, reflector, query, args...)
}

// QueryRowValueOr queries a single value of type T
// or returns the passed defaultVal in case of sql.ErrNoRows.
func QueryRowValueOr[T any](ctx context.Context, defaultVal T, query string, args ...any) (val T, err error) {
	var (
		conn      = Conn(ctx)
		reflector = GetStructReflector(ctx)
	)
	return sqldb.QueryRowValueOr[T](ctx, conn, reflector, defaultVal, query, args...)
}

func QueryRowValueStmt[T any](ctx context.Context, query string) (queryFunc func(ctx context.Context, args ...any) (T, error), closeStmt func() error, err error) {
	var (
		conn      = Conn(ctx)
		reflector = GetStructReflector(ctx)
	)
	return sqldb.QueryRowValueStmt[T](ctx, conn, reflector, query)
}

// ReadRowStruct uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type `*S` that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
func ReadRowStruct[S sqldb.StructWithTableName](ctx context.Context, pkValue any, pkValues ...any) (S, error) {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.ReadRowStruct[S](ctx, conn, queryBuilder, reflector, pkValue, pkValues...)
}

// ReadRowStructOr uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValue+pkValues.
func ReadRowStructOr[S sqldb.StructWithTableName](ctx context.Context, defaultVal S, pkValue any, pkValues ...any) (S, error) {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.ReadRowStructOr[S](ctx, conn, queryBuilder, reflector, defaultVal, pkValue, pkValues...)
}

// QueryRowsAsSlice returns queried rows as slice of the generic type T
// using the passed reflector to scan column values as struct fields.
// QueryRowsAsSlice returns queried rows as slice of the generic type T.
func QueryRowsAsSlice[T any](ctx context.Context, query string, args ...any) (rows []T, err error) {
	var (
		conn      = Conn(ctx)
		reflector = GetStructReflector(ctx)
	)
	return sqldb.QueryRowsAsSlice[T](ctx, conn, reflector, query, args...)
}

// QueryRowsAsStrings scans the query result into a table of strings
// where the first row is a header row with the column names.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
//
// If the query result has no rows, then only the header row
// and no error will be returned.
func QueryRowsAsStrings(ctx context.Context, query string, args ...any) (rows [][]string, err error) {
	var (
		conn = Conn(ctx)
	)
	return sqldb.QueryRowsAsStrings(ctx, conn, query, args...)
}

// QueryCallback calls the passed callback
// with scanned values or a struct for every row.
//
// If the callback function has a single struct or struct pointer argument,
// then RowScanner.ScanStruct will be used per row,
// else RowScanner.Scan will be used for all arguments of the callback.
// If the function has a context.Context as first argument,
// then the passed ctx will be passed on.
//
// The callback can have no result or a single error result value.
//
// If a non nil error is returned from the callback, then this error
// is returned immediately by this function without scanning further rows.
//
// In case of zero rows, no error will be returned.
func QueryCallback(ctx context.Context, callback any, query string, args ...any) error {
	var (
		conn      = Conn(ctx)
		reflector = GetStructReflector(ctx)
	)
	return sqldb.QueryCallback(ctx, conn, reflector, callback, query, args...)
}
