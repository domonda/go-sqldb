package pqconn

import (
	"context"
	"database/sql"

	"github.com/domonda/go-sqldb"
)

type stmt struct {
	query string
	std   *sql.Stmt
}

func (s stmt) PreparedQuery() string {
	return s.query
}

func (s stmt) Exec(ctx context.Context, args ...any) error {
	wrapArrayArgs(args)
	_, err := s.std.ExecContext(ctx, args...)
	return wrapKnownErrors(err)
}

func (s stmt) ExecRowsAffected(ctx context.Context, args ...any) (int64, error) {
	wrapArrayArgs(args)
	result, err := s.std.ExecContext(ctx, args...)
	if err != nil {
		return 0, wrapKnownErrors(err)
	}
	return result.RowsAffected()
}

func (s stmt) Query(ctx context.Context, args ...any) sqldb.Rows {
	wrapArrayArgs(args)
	sqlRows, err := s.std.QueryContext(ctx, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	return rows{sqlRows}
}

func (s stmt) Close() error {
	return s.std.Close()
}
