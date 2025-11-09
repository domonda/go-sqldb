package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Prepare a statement for execution
// with the given query string.
func Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	config := Conn(ctx)
	stmt, err := config.Connection.Prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	return stmtWithErrWrapping{stmt, config.QueryFormatter}, nil
}

type stmtWithErrWrapping struct {
	sqldb.Stmt
	fmt sqldb.QueryFormatter
}

func (s stmtWithErrWrapping) Exec(ctx context.Context, args ...any) error {
	err := s.Stmt.Exec(ctx, args...)
	if err != nil {
		return sqldb.WrapErrorWithQuery(err, s.PreparedQuery(), args, s.fmt)
	}
	return nil
}

func (s stmtWithErrWrapping) Query(ctx context.Context, args ...any) sqldb.Rows {
	rows := s.Stmt.Query(ctx, args...)
	if rows.Err() != nil {
		return sqldb.NewErrRows(sqldb.WrapErrorWithQuery(rows.Err(), s.PreparedQuery(), args, s.fmt))
	}
	return rows
}
