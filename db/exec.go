package db

import "context"

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	conn := Conn(ctx)
	err := conn.Exec(ctx, query, args...)
	if err != nil {
		return wrapErrorWithQuery(err, query, args, conn)
	}
	return nil
}

// ExecStmt returns a function that can be used to execute a prepared statement
// with optional args.
func ExecStmt(ctx context.Context, query string) (execFunc func(ctx context.Context, args ...any) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, wrapErrorWithQuery(err, query, nil, conn)
	}
	execFunc = func(ctx context.Context, args ...any) error {
		err := stmt.Exec(ctx, args...)
		if err != nil {
			return wrapErrorWithQuery(err, query, args, conn)
		}
		return nil
	}
	return execFunc, stmt.Close, nil
}
