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

func (e *MockConn) FormatTableName(name string) (string, error) {
	if e.QueryFormatter == nil {
		return StdQueryFormatter{}.FormatTableName(name)
	}
	return e.QueryFormatter.FormatTableName(name)
}

func (e *MockConn) FormatColumnName(name string) (string, error) {
	if e.QueryFormatter == nil {
		return StdQueryFormatter{}.FormatColumnName(name)
	}
	return e.QueryFormatter.FormatColumnName(name)
}

func (e *MockConn) FormatPlaceholder(paramIndex int) string {
	if e.QueryFormatter == nil {
		return StdQueryFormatter{}.FormatPlaceholder(paramIndex)
	}
	return e.QueryFormatter.FormatPlaceholder(paramIndex)
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
		return c.TxNo, nil
	}
	return c.MockTransactionInfo()
}

func (e *MockConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	if e.MockBegin == nil {
		tx := *e // copy
		tx.TxNo = no
		return &tx, nil
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

func NewRecordingMockConn(placeholderFmt string, normalize bool) *RecordingMockConn {
	return &RecordingMockConn{
		MockConn: MockConn{
			QueryFormatter: StdQueryFormatter{PlaceholderFmt: placeholderFmt},
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
