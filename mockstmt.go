package sqldb

import (
	"context"
)

var _ Stmt = new(MockStmt)

// MockStmt implements the Stmt interface for testing purposes.
// Methods where the corresponding mock function is nil
// return sane defaults (context error for Exec/Query, nil for Close).
type MockStmt struct {
	Prepared  string
	MockExec  func(ctx context.Context, args ...any) error
	MockQuery func(ctx context.Context, args ...any) Rows
	MockClose func() error
}

// PreparedQuery returns the prepared query string.
func (s *MockStmt) PreparedQuery() string {
	return s.Prepared
}

// Exec implements Stmt by calling MockExec
// or returning the context error if MockExec is nil.
func (s *MockStmt) Exec(ctx context.Context, args ...any) error {
	if s.MockExec == nil {
		return ctx.Err()
	}
	return s.MockExec(ctx, args...)
}

// Query implements Stmt by calling MockQuery
// or returning ErrRows with the context error if MockQuery is nil.
func (s *MockStmt) Query(ctx context.Context, args ...any) Rows {
	if s.MockQuery == nil {
		return NewErrRows(ctx.Err())
	}
	return s.MockQuery(ctx, args...)
}

// Close implements Stmt by calling MockClose
// or returning nil if MockClose is nil.
func (s *MockStmt) Close() error {
	if s.MockClose == nil {
		return nil
	}
	return s.MockClose()
}
