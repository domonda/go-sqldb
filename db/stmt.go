package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

func Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	conn := Conn(ctx)
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	return wrapErrStmt{stmt, conn}, nil
}

type wrapErrStmt struct {
	sqldb.Stmt
	fmt sqldb.QueryFormatter
}

func (s wrapErrStmt) Exec(ctx context.Context, args ...any) error {
	err := s.Stmt.Exec(ctx, args...)
	if err != nil {
		return wrapErrorWithQuery(err, s.PreparedQuery(), args, s.fmt)
	}
	return nil
}

func (s wrapErrStmt) Query(ctx context.Context, args ...any) sqldb.Rows {
	rows := s.Stmt.Query(ctx, args...)
	if rows.Err() != nil {
		return sqldb.NewErrRows(wrapErrorWithQuery(rows.Err(), s.PreparedQuery(), args, s.fmt))
	}
	return rows
}
