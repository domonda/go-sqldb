package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
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
	t, err := QueryValue[time.Time](ctx, `SELECT CURRENT_TIMESTAMP`)
	if err != nil {
		return time.Now()
	}
	return t
}

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	conn := Conn(ctx)
	err := conn.Exec(ctx, query, args...)
	if err != nil {
		return wrapErrorWithQuery(err, query, args, conn)
	}
	return nil
}

// QueryRow queries a single row and returns a RowScanner for the results.
func QueryRow(ctx context.Context, query string, args ...any) *RowScanner {
	conn := Conn(ctx)
	rows := conn.Query(ctx, query, args...)
	return NewRowScanner(rows, DefaultStructReflector, conn, query, args)
}

// // QueryRows queries multiple rows and returns a RowsScanner for the results.
// func QueryRows(ctx context.Context, query string, args ...any) *MultiRowScanner {
// 	conn := Conn(ctx)
// 	rows := conn.Query(query, args...)
// 	return NewMultiRowScanner(ctx, rows, DefaultStructReflector, conn, query, args)
// }

// QueryValue queries a single value of type T.
func QueryValue[T any](ctx context.Context, query string, args ...any) (value T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		return *new(T), err
	}
	return value, nil
}

// QueryValueReplaceErrNoRows queries a single value of type T.
// In case of an sql.ErrNoRows error, errNoRows will be called
// and its result returned together with the default value for T.
func QueryValueReplaceErrNoRows[T any](ctx context.Context, errNoRows func() error, query string, args ...any) (value T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && errNoRows != nil {
			return *new(T), errNoRows()
		}
		return *new(T), err
	}
	return value, nil
}

// QueryValueOr queries a single value of type T
// or returns the passed defaultValue in case of sql.ErrNoRows.
func QueryValueOr[T any](ctx context.Context, defaultValue T, query string, args ...any) (value T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultValue, nil
		}
		return *new(T), err
	}
	return value, err
}

// QueryRowStruct queries a row and scans it as struct.
func QueryRowStruct[S any](ctx context.Context, query string, args ...any) (row *S, err error) {
	conn := Conn(ctx)
	rows := conn.Query(ctx, query, args...)
	defer rows.Close()
	err = scanStruct(rows, DefaultStructReflector, &row)
	if err != nil {
		return nil, wrapErrorWithQuery(err, query, args, conn)
	}
	return row, nil
}

// QueryRowStructReplaceErrNoRows queries a row and scans it as struct.
// In case of an sql.ErrNoRows error, errNoRows will be called
// and its result returned as error together with nil as row.
func QueryRowStructReplaceErrNoRows[S any](ctx context.Context, errNoRows func() error, query string, args ...any) (row *S, err error) {
	row, err = QueryRowStruct[S](ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && errNoRows != nil {
			return nil, errNoRows()
		}
		return nil, err
	}
	return row, nil
}

// QueryRowStructOrNil queries a row and scans it as struct
// or returns nil in case of sql.ErrNoRows.
func QueryRowStructOrNil[S any](ctx context.Context, query string, args ...any) (row *S, err error) {
	row, err = QueryRowStruct[S](ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetRow uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
func GetRow[S StructWithTableName](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	// Using explicit first pkValue value
	// to not be able to compile without any value
	pkValues = append([]any{pkValue}, pkValues...)
	t := reflect.TypeOf(row).Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct template type instead of %s", t)
	}
	conn := Conn(ctx)
	table, err := DefaultStructReflector.TableNameForStruct(t)
	if err != nil {
		return nil, err
	}
	table, err = conn.FormatTableName(table)
	if err != nil {
		return nil, err
	}
	pkColumns, err := pkColumnsOfStruct(DefaultStructReflector, t)
	if err != nil {
		return nil, err
	}
	if len(pkColumns) != len(pkValues) {
		return nil, fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	for i, column := range pkColumns {
		pkColumns[i], err = conn.FormatColumnName(column)
		if err != nil {
			return nil, err
		}
	}
	var query strings.Builder
	fmt.Fprintf(&query, `SELECT * FROM %s WHERE %s = %s`, table, pkColumns[0], conn.FormatPlaceholder(0)) //#nosec G104
	for i := 1; i < len(pkColumns); i++ {
		fmt.Fprintf(&query, ` AND %s = %s`, pkColumns[i], conn.FormatPlaceholder(i)) //#nosec G104
	}
	return QueryRowStruct[S](ctx, query.String(), pkValues...)
}

// GetRowOrNil uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValue+pkValues.
func GetRowOrNil[S StructWithTableName](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	row, err = GetRow[S](ctx, pkValue, pkValues...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// QueryStructSlice returns queried rows as slice of the generic type S
// which must be a struct or a pointer to a struct.
func QueryStructSlice[S any](ctx context.Context, query string, args ...any) (rows []S, err error) {
	conn := Conn(ctx)
	sqlRows := conn.Query(ctx, query, args...)
	err = ScanRowsAsSlice(ctx, sqlRows, DefaultStructReflector, &rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
