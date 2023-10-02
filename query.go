package sqldb

import (
	"context"
	"time"
)

func ContextConnection(ctx context.Context) Connection {
	panic("TODO")
}

// Now returns the result of the SQL now()
// function for the current connection.
// Useful for getting the timestamp of a
// SQL transaction for use in Go code.
func Now(ctx context.Context) (time.Time, error) {
	panic("TODO")
}

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	conn := ContextConnection(ctx)
	err := conn.Exec(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, conn)
	}
	return nil
}
