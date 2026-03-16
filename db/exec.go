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

// ExecStmt returns a function that can be used to execute a prepared statement
// with optional args.
func ExecStmt(ctx context.Context, query string) (execFunc func(ctx context.Context, args ...any) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.ExecStmt(ctx, conn, conn, query)
}
