package sqldb

import (
	"context"
	"database/sql"
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
//
// If QueryFormatter is nil, StdQueryFormatter is used.
//
// If TxNo is returned by TransactionInfo
// and a non zero value simulates a transaction.
type MockConn struct {
	QueryFormatter        // StdQueryFormatter{} is used if nil
	TxNo           uint64 // Returned by TransactionInfo

	MockPing                 func(context.Context, time.Duration) error
	MockStats                func() sql.DBStats
	MockConfig               func() *Config
	MockExec                 func(ctx context.Context, query string, args ...any) error
	MockQuery                func(ctx context.Context, query string, args ...any) Rows
	MockPrepare              func(ctx context.Context, query string) (Stmt, error)
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

func (c *MockConn) FormatTableName(name string) (string, error) {
	if c.QueryFormatter == nil {
		return StdQueryFormatter{}.FormatTableName(name)
	}
	return c.QueryFormatter.FormatTableName(name)
}

func (c *MockConn) FormatColumnName(name string) (string, error) {
	if c.QueryFormatter == nil {
		return StdQueryFormatter{}.FormatColumnName(name)
	}
	return c.QueryFormatter.FormatColumnName(name)
}

func (c *MockConn) FormatPlaceholder(paramIndex int) string {
	if c.QueryFormatter == nil {
		return StdQueryFormatter{}.FormatPlaceholder(paramIndex)
	}
	return c.QueryFormatter.FormatPlaceholder(paramIndex)
}

func (c *MockConn) Ping(ctx context.Context, timeout time.Duration) error {
	if c.MockPing == nil {
		return ctx.Err()
	}
	return c.MockPing(ctx, timeout)
}

func (c *MockConn) Stats() sql.DBStats {
	if c.MockPing == nil {
		return sql.DBStats{}
	}
	return c.MockStats()
}

func (c *MockConn) Config() *Config {
	if c.MockConfig == nil {
		return &Config{Driver: "MockConn"}
	}
	return c.MockConfig()
}

func (c *MockConn) Exec(ctx context.Context, query string, args ...any) error {
	if c.MockExec == nil {
		return ctx.Err()
	}
	return c.MockExec(ctx, query, args...)
}

func (c *MockConn) Query(ctx context.Context, query string, args ...any) Rows {
	if c.MockQuery == nil {
		return NewErrRows(ctx.Err())
	}
	return c.MockQuery(ctx, query, args...)
}

func (c *MockConn) Prepare(ctx context.Context, query string) (Stmt, error) {
	if c.MockPrepare == nil {
		return &MockStmt{
			Prepared: query,
			MockExec: func(ctx context.Context, args ...any) error {
				return c.Exec(ctx, query, args...)
			},
			MockQuery: func(ctx context.Context, args ...any) Rows {
				return c.Query(ctx, query, args...)
			},
		}, ctx.Err()
	}
	return c.MockPrepare(ctx, query)
}

func (c *MockConn) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	if c.MockTransactionInfo == nil {
		return c.TxNo, nil
	}
	return c.MockTransactionInfo()
}

func (c *MockConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	if c.MockBegin == nil {
		tx := *c // copy
		tx.TxNo = no
		return &tx, nil
	}
	return c.MockBegin(ctx, no, opts)
}

func (c *MockConn) Commit() error {
	if c.MockCommit == nil {
		return nil
	}
	return c.MockCommit()
}

func (c *MockConn) Rollback() error {
	if c.MockCommit == nil {
		return nil
	}
	return c.MockCommit()
}

func (c *MockConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	if c.MockListenOnChannel == nil {
		return nil
	}
	return c.MockListenOnChannel(channel, onNotify, onUnlisten)
}

func (c *MockConn) UnlistenChannel(channel string) error {
	if c.MockUnlistenChannel == nil {
		return nil
	}
	return c.MockUnlistenChannel(channel)
}

func (c *MockConn) IsListeningOnChannel(channel string) bool {
	if c.MockIsListeningOnChannel == nil {
		return false
	}
	return c.MockIsListeningOnChannel(channel)
}

func (c *MockConn) Close() error {
	if c.MockClose == nil {
		return nil
	}
	return c.MockClose()
}

// ----------------------------------------------------------------------------

type MockConnRecording struct {
	Execs   []QueryData
	Queries []QueryData
}

type RecordingMockConn struct {
	MockConn
	MockConnRecording
	Normalize bool
}

func NewRecordingMockConn(placeholderPosPrefix string, normalize bool) *RecordingMockConn {
	return &RecordingMockConn{
		MockConn: MockConn{
			QueryFormatter: StdQueryFormatter{PlaceholderPosPrefix: placeholderPosPrefix},
		},
		Normalize: normalize,
	}
}

func (c *RecordingMockConn) Exec(ctx context.Context, query string, args ...any) error {
	queryData, err := NewQueryData(query, args, c.Normalize)
	if err != nil {
		return err
	}
	c.Execs = append(c.Execs, queryData)
	return c.MockConn.Exec(ctx, query, args...)
}

func (c *RecordingMockConn) Query(ctx context.Context, query string, args ...any) Rows {
	queryData, err := NewQueryData(query, args, c.Normalize)
	if err != nil {
		return NewErrRows(err)
	}
	c.Queries = append(c.Queries, queryData)
	return c.MockConn.Query(ctx, query, args...)
}
