package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	return sqldb.Exec(ctx, Conn(ctx), query, args...)
}

// ExecStmt returns a function that can be used to execute a prepared statement
// with optional args.
func ExecStmt(ctx context.Context, query string) (execFunc func(ctx context.Context, args ...any) error, closeStmt func() error, err error) {
	return sqldb.ExecStmt(ctx, Conn(ctx), query)
}
