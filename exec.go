package sqldb

import "context"

// Exec executes a query with optional args.
func Exec(ctx context.Context, c *ConnExt, query string, args ...any) error {
	err := c.Exec(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, c.QueryFormatter)
	}
	return nil
}

// ExecStmt returns a function that can be used to execute a prepared statement
// with optional args.
func ExecStmt(ctx context.Context, c *ConnExt, query string) (execFunc func(ctx context.Context, args ...any) error, closeStmt func() error, err error) {
	stmt, err := c.Prepare(ctx, query)
	if err != nil {
		return nil, nil, WrapErrorWithQuery(err, query, nil, c.QueryFormatter)
	}
	execFunc = func(ctx context.Context, args ...any) error {
		err := stmt.Exec(ctx, args...)
		if err != nil {
			return WrapErrorWithQuery(err, query, args, c.QueryFormatter)
		}
		return nil
	}
	return execFunc, stmt.Close, nil
}
