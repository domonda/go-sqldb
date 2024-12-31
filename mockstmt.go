package sqldb

import (
	"context"
)

type MockStmt struct {
	Prepared  string
	MockExec  func(ctx context.Context, args ...any) error
	MockQuery func(ctx context.Context, args ...any) Rows
	MockClose func() error
}

func (s *MockStmt) PreparedQuery() string {
	return s.Prepared
}

func (s *MockStmt) Exec(ctx context.Context, args ...any) error {
	if s.MockExec == nil {
		return ctx.Err()
	}
	return s.MockExec(ctx, args...)
}

func (s *MockStmt) Query(ctx context.Context, args ...any) Rows {
	if s.MockQuery == nil {
		return NewErrRows(ctx.Err())
	}
	return s.MockQuery(ctx, args...)
}

func (s *MockStmt) Close() error {
	if s.MockClose == nil {
		return nil
	}
	return s.MockClose()
}
