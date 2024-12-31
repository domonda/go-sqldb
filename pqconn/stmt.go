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

func (s stmt) Query(ctx context.Context, args ...any) sqldb.Rows {
	wrapArrayArgs(args)
	rows, err := s.std.QueryContext(ctx, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	return rows
}

func (s stmt) Close() error {
	return s.std.Close()
}
