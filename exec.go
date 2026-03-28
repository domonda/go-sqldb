package sqldb

import "context"

// Exec executes a query with optional args.
func Exec(ctx context.Context, conn Executor, fmtr QueryFormatter, query string, args ...any) error {
	err := conn.Exec(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, fmtr)
	}
	return nil
}

// ExecRowsAffected executes a query with optional args
// and returns the number of rows affected by an
// update, insert, or delete. Not every database or database
// driver may support this.
func ExecRowsAffected(ctx context.Context, conn Executor, fmtr QueryFormatter, query string, args ...any) (int64, error) {
	n, err := conn.ExecRowsAffected(ctx, query, args...)
	if err != nil {
		return 0, WrapErrorWithQuery(err, query, args, fmtr)
	}
	return n, nil
}

// ExecStmt returns a function that can be used to execute a prepared statement
// with optional args.
func ExecStmt(ctx context.Context, conn Preparer, fmtr QueryFormatter, query string) (execFunc func(ctx context.Context, args ...any) error, closeStmt func() error, err error) {
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, WrapErrorWithQuery(err, query, nil, fmtr)
	}
	execFunc = func(ctx context.Context, args ...any) error {
		err := stmt.Exec(ctx, args...)
		if err != nil {
			return WrapErrorWithQuery(err, query, args, fmtr)
		}
		return nil
	}
	return execFunc, stmt.Close, nil
}

// ExecRowsAffectedStmt returns a function that can be used to execute
// a prepared statement with optional args and returns the number of
// rows affected by an update, insert, or delete.
func ExecRowsAffectedStmt(ctx context.Context, conn Preparer, fmtr QueryFormatter, query string) (execFunc func(ctx context.Context, args ...any) (int64, error), closeStmt func() error, err error) {
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, WrapErrorWithQuery(err, query, nil, fmtr)
	}
	execFunc = func(ctx context.Context, args ...any) (int64, error) {
		n, err := stmt.ExecRowsAffected(ctx, args...)
		if err != nil {
			return 0, WrapErrorWithQuery(err, query, args, fmtr)
		}
		return n, nil
	}
	return execFunc, stmt.Close, nil
}
