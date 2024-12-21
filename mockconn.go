package sqldb

import (
	"context"
	"database/sql"
	"strconv"
	"time"
)

// MockConn implements ListenerConnection
var _ ListenerConnection = new(MockConn)

// MockConn implements the ListenerConnection interface
// with mock fuctions for testing.
//
// Methods where the corresponding mock function is nil
// return sane defaults and no errors,
// exept for methods with a context argument
// where the context error is returned.
type MockConn struct {
	MockPing                 func(context.Context, time.Duration) error
	MockStats                func() sql.DBStats
	MockConfig               func() *Config
	MockPlaceholder          func(paramIndex int) string
	MockValidateColumnName   func(name string) error
	MockExec                 func(ctx context.Context, query string, args ...any) error
	MockQuery                func(ctx context.Context, query string, args ...any) Rows
	MockTransactionInfo      func() (no uint64, opts *sql.TxOptions)
	MockBegin                func(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error)
	MockCommit               func() error
	MockRollback             func() error
	MockTransaction          func(opts *sql.TxOptions, txFunc func(tx Connection) error) error
	MockListenOnChannel      func(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error
	MockUnlistenChannel      func(channel string) error
	MockIsListeningOnChannel func(channel string) bool
	MockClose                func() error
}

func (e *MockConn) Ping(ctx context.Context, timeout time.Duration) error {
	if e.MockPing == nil {
		return ctx.Err()
	}
	return e.MockPing(ctx, timeout)
}

func (e *MockConn) Stats() sql.DBStats {
	if e.MockPing == nil {
		return sql.DBStats{}
	}
	return e.MockStats()
}

func (e *MockConn) Config() *Config {
	if e.MockConfig == nil {
		return &Config{Driver: "MockConn"}
	}
	return e.MockConfig()
}

func (e *MockConn) Placeholder(paramIndex int) string {
	if e.MockPlaceholder == nil {
		return "?" + strconv.Itoa(paramIndex+1)
	}
	return e.MockPlaceholder(paramIndex)
}

func (e *MockConn) ValidateColumnName(name string) error {
	if e.MockValidateColumnName == nil {
		return nil
	}
	return e.MockValidateColumnName(name)
}

func (e *MockConn) Exec(ctx context.Context, query string, args ...any) error {
	if e.MockExec == nil {
		return ctx.Err()
	}
	return e.MockExec(ctx, query, args...)
}

func (e *MockConn) Query(ctx context.Context, query string, args ...any) Rows {
	if e.MockQuery == nil {
		return NewErrRows(ctx.Err())
	}
	return e.MockQuery(ctx, query, args...)
}

func (c *MockConn) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	if c.MockTransactionInfo == nil {
		return 0, nil
	}
	return c.MockTransactionInfo()
}

func (e *MockConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	if e.MockBegin == nil {
		return e, nil
	}
	return e.MockBegin(ctx, no, opts)
}

func (e *MockConn) Commit() error {
	if e.MockCommit == nil {
		return nil
	}
	return e.MockCommit()
}

func (e *MockConn) Rollback() error {
	if e.MockCommit == nil {
		return nil
	}
	return e.MockCommit()
}

func (e *MockConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	if e.MockListenOnChannel == nil {
		return nil
	}
	return e.MockListenOnChannel(channel, onNotify, onUnlisten)
}

func (e *MockConn) UnlistenChannel(channel string) error {
	if e.MockUnlistenChannel == nil {
		return nil
	}
	return e.MockUnlistenChannel(channel)
}

func (e *MockConn) IsListeningOnChannel(channel string) bool {
	if e.MockIsListeningOnChannel == nil {
		return false
	}
	return e.MockIsListeningOnChannel(channel)
}

func (e *MockConn) Close() error {
	if e.MockClose == nil {
		return nil
	}
	return e.MockClose()
}
