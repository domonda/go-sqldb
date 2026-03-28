package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	conn := Conn(ctx)
	return sqldb.Exec(ctx, conn, conn, query, args...)
}

// ExecRowsAffected executes a query with optional args
// and returns the number of rows affected by an
// update, insert, or delete. Not every database or database
// driver may support this.
func ExecRowsAffected(ctx context.Context, query string, args ...any) (int64, error) {
	conn := Conn(ctx)
	return sqldb.ExecRowsAffected(ctx, conn, conn, query, args...)
}

// ExecStmt returns a function that can be used to execute a prepared statement
// with optional args.
func ExecStmt(ctx context.Context, query string) (execFunc func(ctx context.Context, args ...any) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.ExecStmt(ctx, conn, conn, query)
}

// ExecRowsAffectedStmt returns a function that can be used to execute
// a prepared statement with optional args and returns the number of
// rows affected by an update, insert, or delete.
func ExecRowsAffectedStmt(ctx context.Context, query string) (execFunc func(ctx context.Context, args ...any) (int64, error), closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.ExecRowsAffectedStmt(ctx, conn, conn, query)
}
