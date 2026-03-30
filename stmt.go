package sqldb

import (
	"context"
	"database/sql"
	"net"
)

// Stmt is a prepared or unprepared SQL statement
// that can be executed multiple times with different arguments.
type Stmt interface {
	PreparedQuery() string
	Exec(ctx context.Context, args ...any) error
	ExecRowsAffected(ctx context.Context, args ...any) (int64, error)
	Query(ctx context.Context, args ...any) Rows
	Close() error
}

type wrappedStmt struct {
	stmt    *sql.Stmt
	query   string
	wrapErr func(error) error
}

// NewStmt returns a Stmt wrapping a [*sql.Stmt] with the query string
// that was used to prepare the statement.
// The optional wrapErr function is called on non-nil errors
// to wrap them with driver-specific known error classifications.
func NewStmt(stmt *sql.Stmt, query string, wrapErr func(error) error) Stmt {
	return wrappedStmt{stmt: stmt, query: query, wrapErr: wrapErr}
}

func (s wrappedStmt) PreparedQuery() string {
	return s.query
}

func (s wrappedStmt) wrapError(err error) error {
	if err != nil && s.wrapErr != nil {
		return s.wrapErr(err)
	}
	return err
}

func (s wrappedStmt) Exec(ctx context.Context, args ...any) error {
	_, err := s.stmt.ExecContext(ctx, args...)
	return s.wrapError(err)
}

func (s wrappedStmt) ExecRowsAffected(ctx context.Context, args ...any) (int64, error) {
	result, err := s.stmt.ExecContext(ctx, args...)
	if err != nil {
		return 0, s.wrapError(err)
	}
	return result.RowsAffected()
}

func (s wrappedStmt) Query(ctx context.Context, args ...any) Rows {
	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		return NewErrRows(s.wrapError(err))
	}
	return rows
}

func (s wrappedStmt) Close() error {
	return s.stmt.Close()
}

type unpreparedStmt struct {
	conn      Connection
	query     string
	closeFunc func() error
}

// NewUnpreparedStmt returns a Stmt that executes the query
// on the given Connection each time without preparing it first.
// The optional closeFunc is called when the Stmt is closed.
func NewUnpreparedStmt(conn Connection, query string, closeFunc func() error) Stmt {
	return &unpreparedStmt{conn, query, closeFunc}
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

func (s *unpreparedStmt) ExecRowsAffected(ctx context.Context, args ...any) (int64, error) {
	if s.conn == nil {
		return 0, net.ErrClosed
	}
	return s.conn.ExecRowsAffected(ctx, s.query, args...)
}

func (s *unpreparedStmt) Query(ctx context.Context, args ...any) Rows {
	if s.conn == nil {
		return NewErrRows(net.ErrClosed)
	}
	return s.conn.Query(ctx, s.query, args...)
}

func (s *unpreparedStmt) Close() error {
	s.conn = nil
	if s.closeFunc == nil {
		return nil
	}
	return s.closeFunc()
}
