package db

import (
	"context"
	"time"

	"github.com/domonda/go-sqldb"
)

// Now returns the result of the SQL now()
// function for the current connection.
// Useful for getting the timestamp of a
// SQL transaction for use in Go code.
func Now(ctx context.Context) (time.Time, error) {
	return Conn(ctx).Now()
}

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	return Conn(ctx).Exec(query, args...)
}

// QueryRow queries a single row and returns a RowScanner for the results.
func QueryRow(ctx context.Context, query string, args ...any) sqldb.RowScanner {
	return Conn(ctx).QueryRow(query, args...)
}

// QueryRows queries multiple rows and returns a RowsScanner for the results.
func QueryRows(ctx context.Context, query string, args ...any) sqldb.RowsScanner {
	return Conn(ctx).QueryRows(query, args...)
}

// QueryValue queries a single value of type T.
func QueryValue[T any](ctx context.Context, query string, args ...any) (T, error) {
	var val T
	err := Conn(ctx).QueryRow(query, args...).Scan(&val)
	return val, err
}
