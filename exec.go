package sqldb

import "context"

// Exec executes a query with optional args.
func Exec(ctx context.Context, conn Queryer, query string, args ...any) error {
	err := conn.Exec(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, conn)
	}
	return nil
}

// ExecStmt returns a function that can be used to execute a prepared statement
// with optional args.
func ExecStmt(ctx context.Context, conn Queryer, query string) (execFunc func(ctx context.Context, args ...any) error, closeStmt func() error, err error) {
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, WrapErrorWithQuery(err, query, nil, conn)
	}
	execFunc = func(ctx context.Context, args ...any) error {
		err := stmt.Exec(ctx, args...)
		if err != nil {
			return WrapErrorWithQuery(err, query, args, conn)
		}
		return nil
	}
	return execFunc, stmt.Close, nil
}
