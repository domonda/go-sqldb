package sqldb

import (
	"context"
	"database/sql"
	"net"
)

type Stmt interface {
	PreparedQuery() string
	Exec(ctx context.Context, args ...any) error
	Query(ctx context.Context, args ...any) Rows
	Close() error
}

type wrappedStmt struct {
	stmt  *sql.Stmt
	query string
}

func NewStmt(stmt *sql.Stmt, query string) Stmt {
	return wrappedStmt{stmt: stmt, query: query}
}

func (s wrappedStmt) PreparedQuery() string {
	return s.query
}

func (s wrappedStmt) Exec(ctx context.Context, args ...any) error {
	_, err := s.stmt.ExecContext(ctx, args...)
	return err
}

func (s wrappedStmt) Query(ctx context.Context, args ...any) Rows {
	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		return NewErrRows(err)
	}
	return rows
}

func (s wrappedStmt) Close() error {
	return s.stmt.Close()
}

type unpreparedStmt struct {
	conn  Connection
	query string
}

func NewUnpreparedStmt(conn Connection, query string) Stmt {
	return &unpreparedStmt{conn: conn, query: query}
}

func (s *unpreparedStmt) PreparedQuery() string {
	return s.query
}

func (s *unpreparedStmt) Exec(ctx context.Context, args ...any) error {
	if s.conn == nil {
		return net.ErrClosed
	}
	return s.conn.Exec(ctx, s.query, args...)
}

func (s *unpreparedStmt) Query(ctx context.Context, args ...any) Rows {
	if s.conn == nil {
		return NewErrRows(net.ErrClosed)
	}
	return s.conn.Query(ctx, s.query, args...)
}

func (s *unpreparedStmt) Close() error {
	s.conn = nil
	return nil
}
