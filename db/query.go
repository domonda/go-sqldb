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
	t, err := QueryRowAs[time.Time](ctx,
		/*sql*/ `SELECT CURRENT_TIMESTAMP`,
	)
	if err != nil {
		return time.Now()
	}
	return t
}

// QueryRow queries a single row and returns a Row for the results.
func QueryRow(ctx context.Context, query string, args ...any) *sqldb.Row {
	conn := Conn(ctx)
	return sqldb.QueryRow(
		ctx,
		conn,
		StructReflector(ctx),
		conn,
		query,
		args...,
	)
}

// QueryRowAs queries a single row and scans it as the type T.
// If T is a struct that does not implement sql.Scanner,
// the column values are scanned into the struct fields.
func QueryRowAs[T any](ctx context.Context, query string, args ...any) (val T, err error) {
	conn := Conn(ctx)
	return sqldb.QueryRowAs[T](
		ctx,
		conn,
		StructReflector(ctx),
		conn,
		query,
		args...,
	)
}

// QueryRowAs2 queries a single row and scans it into 2 typed values.
func QueryRowAs2[T0, T1 any](ctx context.Context, query string, args ...any) (val0 T0, val1 T1, err error) {
	err = QueryRow(ctx, query, args...).Scan(&val0, &val1)
	return
}

// QueryRowAs3 queries a single row and scans it into 3 typed values.
func QueryRowAs3[T0, T1, T2 any](ctx context.Context, query string, args ...any) (val0 T0, val1 T1, val2 T2, err error) {
	err = QueryRow(ctx, query, args...).Scan(&val0, &val1, &val2)
	return
}

// QueryRowAs4 queries a single row and scans it into 4 typed values.
func QueryRowAs4[T0, T1, T2, T3 any](ctx context.Context, query string, args ...any) (val0 T0, val1 T1, val2 T2, val3 T3, err error) {
	err = QueryRow(ctx, query, args...).Scan(&val0, &val1, &val2, &val3)
	return
}

// QueryRowAs5 queries a single row and scans it into 5 typed values.
func QueryRowAs5[T0, T1, T2, T3, T4 any](ctx context.Context, query string, args ...any) (val0 T0, val1 T1, val2 T2, val3 T3, val4 T4, err error) {
	err = QueryRow(ctx, query, args...).Scan(&val0, &val1, &val2, &val3, &val4)
	return
}

// QueryRowAsOr queries a single row and scans it as the type T,
// or returns the passed defaultVal in case of sql.ErrNoRows.
func QueryRowAsOr[T any](ctx context.Context, defaultVal T, query string, args ...any) (val T, err error) {
	conn := Conn(ctx)
	return sqldb.QueryRowAsOr(
		ctx,
		conn,
		StructReflector(ctx),
		conn,
		defaultVal,
		query,
		args...,
	)
}

// QueryRowAsStmt prepares a statement that queries a single row and scans it as the type T.
// Returns a queryFunc to execute the query with different args each time
// and a closeStmt function that must be called when done.
func QueryRowAsStmt[T any](ctx context.Context, query string) (queryFunc func(ctx context.Context, args ...any) (T, error), closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.QueryRowAsStmt[T](
		ctx,
		conn,
		StructReflector(ctx),
		conn,
		query,
	)
}

// QueryRowStruct queries a table row by primary key and scans it into a struct of type S.
// Table name and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The number of pkValue+pkValues must match the number of primary key columns.
func QueryRowStruct[S sqldb.StructWithTableName](ctx context.Context, pkValue any, pkValues ...any) (S, error) {
	conn := Conn(ctx)
	return sqldb.QueryRowStruct[S](
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		pkValue,
		pkValues...,
	)
}

// QueryRowStructOr queries a table row by primary key and scans it into a struct of type S.
// Returns defaultVal and no error if no row was found.
// Table name and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., db.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The number of pkValue+pkValues must match the number of primary key columns.
func QueryRowStructOr[S sqldb.StructWithTableName](ctx context.Context, defaultVal S, pkValue any, pkValues ...any) (S, error) {
	conn := Conn(ctx)
	return sqldb.QueryRowStructOr(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		defaultVal,
		pkValue,
		pkValues...,
	)
}

// QueryRowAsMap queries a single row and returns the columns as map
// using the column names as keys.
func QueryRowAsMap[K ~string, V any](ctx context.Context, query string, args ...any) (m map[K]V, err error) {
	conn := Conn(ctx)
	return sqldb.QueryRowAsMap[K, V](
		ctx,
		conn,
		conn,
		query,
		args...,
	)
}

// QueryRowAsStrings queries a single row and returns its column values as strings.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
func QueryRowAsStrings(ctx context.Context, query string, args ...any) ([]string, error) {
	conn := Conn(ctx)
	return sqldb.QueryRowAsStrings(
		ctx,
		conn,
		conn,
		query,
		args...,
	)
}

// QueryRowAsStringsWithHeader queries a single row and returns a [][]string
// where the first slice contains the column names and the second slice
// contains the row values as strings.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
func QueryRowAsStringsWithHeader(ctx context.Context, query string, args ...any) ([][]string, error) {
	conn := Conn(ctx)
	return sqldb.QueryRowAsStringsWithHeader(
		ctx,
		conn,
		conn,
		query,
		args...,
	)
}

// QueryRowsAsMapSlice queries rows and returns them as a slice of maps
// keyed by column name. The values are exactly how they are passed
// from the database driver to an [sql.Scanner]. Byte slices will be copied.
//
// If converter is not nil, it is applied to each scanned value and
// replaces the value in the returned map when it reports a successful
// conversion. Multiple converters can be combined by passing a
// [ScanConverters] slice.
//
// Use this as the multi-row counterpart of [sqldb.Row.ScanMap],
// for example to encode a query result as a JSON array.
func QueryRowsAsMapSlice(ctx context.Context, converter ScanConverter, query string, args ...any) ([]map[string]any, error) {
	conn := Conn(ctx)
	return sqldb.QueryRowsAsMapSlice(
		ctx,
		conn,
		conn,
		converter,
		MaxNumRowsFromContext(ctx),
		query,
		args...,
	)
}

// QueryRowsAsSlice returns queried rows as slice of the generic type T.
// If T is a struct, column values are scanned into fields
// using the [StructReflector] from the context.
//
// The maximum number of rows is read from the context via
// [MaxNumRowsFromContext]; use [ContextWithMaxNumRows] to set it.
// A negative value (the default) disables the limit. A value of 0 is
// enforced as a hard cap that permits no rows. Exceeding the cap returns
// [ErrMaxNumRowsExceeded] along with the rows scanned so far.
//
// On any error the already-scanned rows are returned together with the error.
func QueryRowsAsSlice[T any](ctx context.Context, query string, args ...any) (rows []T, err error) {
	conn := Conn(ctx)
	return sqldb.QueryRowsAsSlice[T](
		ctx,
		conn,
		StructReflector(ctx),
		conn,
		MaxNumRowsFromContext(ctx),
		query,
		args...,
	)
}

// QueryRowsAsStrings scans the query result into a table of strings
// where the first row is a header row with the column names.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
//
// The maximum number of data rows (rows[0] is the header and is not counted)
// is read from the context via [MaxNumRowsFromContext]; use
// [ContextWithMaxNumRows] to set it. A negative value (the default) disables
// the limit. A value of 0 is enforced as a hard cap that permits no data rows.
// Exceeding the cap returns [ErrMaxNumRowsExceeded] along with the
// header and the data rows scanned so far.
//
// If the query result has no rows, then only the header row
// and no error will be returned.
func QueryRowsAsStrings(ctx context.Context, query string, args ...any) (rows [][]string, err error) {
	conn := Conn(ctx)
	return sqldb.QueryRowsAsStrings(
		ctx,
		conn,
		conn,
		MaxNumRowsFromContext(ctx),
		query,
		args...,
	)
}

// QueryStructCallback calls the passed callback with a scanned struct
// for every row returned by the query.
// S must be a struct or pointer to struct type.
// Column values are scanned into struct fields
// using the [StructReflector] from the context.
//
// If a non nil error is returned from the callback, then this error
// is returned immediately without scanning further rows.
//
// In case of zero rows, no error will be returned.
func QueryStructCallback[S any](ctx context.Context, callback func(S) error, query string, args ...any) error {
	conn := Conn(ctx)
	return sqldb.QueryStructCallback(
		ctx,
		conn,
		StructReflector(ctx),
		conn,
		callback,
		query,
		args...,
	)
}

// QueryCallback calls the passed callback
// with scanned values or a struct for every row.
// Struct arguments are scanned using the [StructReflector] from the context.
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
	conn := Conn(ctx)
	return sqldb.QueryCallback(
		ctx,
		conn,
		StructReflector(ctx),
		conn,
		callback,
		query,
		args...,
	)
}
