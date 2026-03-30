package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

// MockConn implements ListenerConnection and QueryFormatter
var (
	_ ListenerConnection = new(MockConn)
	_ QueryFormatter     = new(MockConn)
)

// QueryRecordings holds the recorded exec and query calls
// made through a MockConn.
type QueryRecordings struct {
	Execs   []QueryData
	Queries []QueryData
}

// MockConn implements the ListenerConnection interface
// with mock functions for testing.
// Its methods are safe for concurrent use, simulating the
// thread-safety of real database connections.
// Returned rows from queries are not safe for concurrent use,
// consistent with the standard library's sql.Rows.
//
// Exported struct fields are not protected by the mutex
// and must only be set during setup before concurrent use.
//
// Methods where the corresponding mock function is nil
// return sane defaults and no errors,
// except for methods with a context argument
// where the context error is returned.
//
// If QueryFormatter is nil, StdQueryFormatter is used.
//
// If NormalizeQuery is nil, the query is not normalized.
//
// TxID is returned by TransactionState
// and a non-zero value simulates a transaction.
type MockConn struct {
	// Configuration
	QueryFormatter QueryFormatter     // StdQueryFormatter{} is used if nil
	NormalizeQuery NormalizeQueryFunc // nil means no normalization
	QueryLog       io.Writer          // nil means no writing of queries
	MockMaxArgs    int                // overrides QueryFormatter.MaxArgs() if > 0

	// Connection state
	TxID        uint64 // Returned by TransactionState
	StmtNo      uint64
	ListeningOn map[string]struct{}

	Recordings       QueryRecordings
	MockQueryResults map[string]Rows

	MockConfig               func() *Config
	MockPing                 func(context.Context, time.Duration) error
	MockStats                func() sql.DBStats
	MockExec                 func(ctx context.Context, query string, args ...any) error
	MockExecRowsAffected     func(ctx context.Context, query string, args ...any) (int64, error)
	MockQuery                func(ctx context.Context, query string, args ...any) Rows
	MockPrepare              func(ctx context.Context, query string) (Stmt, error)
	MockTransaction          func() TransactionState
	MockBegin                func(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error)
	MockCommit               func() error
	MockRollback             func() error
	MockListenOnChannel      func(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error
	MockUnlistenChannel      func(channel string) error
	MockIsListeningOnChannel func(channel string) bool
	MockClose                func() error

	mtx sync.Mutex
}

// NewMockConn returns a new MockConn configured with the given QueryFormatter.
// If queryFormatter is nil, StdQueryFormatter{} is used.
// Use WithNormalizeQuery and WithQueryLog to configure further.
func NewMockConn(queryFormatter QueryFormatter) *MockConn {
	return &MockConn{
		QueryFormatter: queryFormatter,
	}
}

// WithNormalizeQuery returns the MockConn with the given NormalizeQueryFunc set.
func (c *MockConn) WithNormalizeQuery(f NormalizeQueryFunc) *MockConn {
	c.NormalizeQuery = f
	return c
}

// WithQueryLog returns the MockConn with the given query log writer set.
func (c *MockConn) WithQueryLog(w io.Writer) *MockConn {
	c.QueryLog = w
	return c
}

// Clone returns a shallow copy of the MockConn
// with cloned ListeningOn and MockQueryResults maps
// and a new mutex.
func (c *MockConn) Clone() *MockConn {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return &MockConn{
		QueryFormatter: c.QueryFormatter,
		NormalizeQuery: c.NormalizeQuery,
		QueryLog:       c.QueryLog,
		MockMaxArgs:    c.MockMaxArgs,
		TxID:           c.TxID,
		StmtNo:         c.StmtNo,
		ListeningOn:    maps.Clone(c.ListeningOn),
		Recordings: QueryRecordings{
			Execs:   slices.Clone(c.Recordings.Execs),
			Queries: slices.Clone(c.Recordings.Queries),
		},
		MockQueryResults:         maps.Clone(c.MockQueryResults),
		MockConfig:               c.MockConfig,
		MockPing:                 c.MockPing,
		MockStats:                c.MockStats,
		MockExec:                 c.MockExec,
		MockExecRowsAffected:     c.MockExecRowsAffected,
		MockQuery:                c.MockQuery,
		MockPrepare:              c.MockPrepare,
		MockTransaction:          c.MockTransaction,
		MockBegin:                c.MockBegin,
		MockCommit:               c.MockCommit,
		MockRollback:             c.MockRollback,
		MockListenOnChannel:      c.MockListenOnChannel,
		MockUnlistenChannel:      c.MockUnlistenChannel,
		MockIsListeningOnChannel: c.MockIsListeningOnChannel,
		MockClose:                c.MockClose,
	}
}

// getQueryFormatter returns c.QueryFormatter
// or StdQueryFormatter{} if nil.
func (c *MockConn) getQueryFormatter() QueryFormatter {
	if c.QueryFormatter != nil {
		return c.QueryFormatter
	}
	return StdQueryFormatter{}
}

// WithQueryResult returns a clone of the MockConn with an additional
// MockRows result registered for the given query and args.
// The query is normalized and formatted using the connection's
// NormalizeQuery and QueryFormatter before being used as lookup key.
func (c *MockConn) WithQueryResult(columns []string, rows [][]driver.Value, forQuery string, args ...any) *MockConn {
	normQuery := MustNormalizeAndFormatQuery(c.NormalizeQuery, c.getQueryFormatter(), forQuery, args...)

	cc := c.Clone()
	if cc.MockQueryResults == nil {
		cc.MockQueryResults = make(map[string]Rows)
	}
	cc.MockQueryResults[normQuery] = NewMockRows(columns...).WithRows(rows)
	return cc
}

// FormatTableName implements QueryFormatter.
func (c *MockConn) FormatTableName(name string) (string, error) {
	return c.getQueryFormatter().FormatTableName(name)
}

// FormatColumnName implements QueryFormatter.
func (c *MockConn) FormatColumnName(name string) (string, error) {
	return c.getQueryFormatter().FormatColumnName(name)
}

// FormatPlaceholder implements QueryFormatter.
func (c *MockConn) FormatPlaceholder(paramIndex int) string {
	return c.getQueryFormatter().FormatPlaceholder(paramIndex)
}

// FormatStringLiteral implements Connection and QueryFormatter.
func (c *MockConn) FormatStringLiteral(str string) string {
	return c.getQueryFormatter().FormatStringLiteral(str)
}

// MaxArgs implements QueryFormatter.
// Returns MockMaxArgs if > 0, otherwise delegates to the QueryFormatter.
func (c *MockConn) MaxArgs() int {
	if c.MockMaxArgs > 0 {
		return c.MockMaxArgs
	}
	return c.getQueryFormatter().MaxArgs()
}

// Config implements Connection by returning MockConfig()
// or a default Config with Driver "MockConn" if MockConfig is nil.
func (c *MockConn) Config() *Config {
	if c.MockConfig == nil {
		return &Config{
			Driver: "MockConn",
		}
	}
	return c.MockConfig()
}

// Ping implements Connection by calling MockPing
// or returning the context error if MockPing is nil.
func (c *MockConn) Ping(ctx context.Context, timeout time.Duration) error {
	if c.MockPing == nil {
		return ctx.Err()
	}
	return c.MockPing(ctx, timeout)
}

// Stats implements Connection by calling MockStats
// or returning a zero sql.DBStats if MockStats is nil.
func (c *MockConn) Stats() sql.DBStats {
	if c.MockStats == nil {
		return sql.DBStats{}
	}
	return c.MockStats()
}

// Exec implements Connection by recording the query,
// optionally logging it, and then calling MockExec
// or returning the context error if MockExec is nil.
func (c *MockConn) Exec(ctx context.Context, query string, args ...any) (err error) {
	queryFormatter := c.getQueryFormatter()
	queryData, err := NewQueryData(query, args, c.NormalizeQuery)
	if err != nil {
		return err
	}

	c.mtx.Lock()
	c.Recordings.Execs = append(c.Recordings.Execs, queryData)
	c.mtx.Unlock()

	if c.QueryLog != nil {
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprint(c.QueryLog, FormatQuery(queryFormatter, query, args...), ";\n")
		if err != nil {
			return err
		}
	}

	if c.MockExec == nil {
		return ctx.Err()
	}
	return c.MockExec(ctx, query, args...)
}

// ExecRowsAffected implements Connection by recording the query,
// optionally logging it, and then calling MockExecRowsAffected
// or returning 0 and the context error if MockExecRowsAffected is nil.
func (c *MockConn) ExecRowsAffected(ctx context.Context, query string, args ...any) (int64, error) {
	queryFormatter := c.getQueryFormatter()
	queryData, err := NewQueryData(query, args, c.NormalizeQuery)
	if err != nil {
		return 0, err
	}

	c.mtx.Lock()
	c.Recordings.Execs = append(c.Recordings.Execs, queryData)
	c.mtx.Unlock()

	if c.QueryLog != nil {
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return 0, err
			}
		}
		_, err = fmt.Fprint(c.QueryLog, FormatQuery(queryFormatter, query, args...), ";\n")
		if err != nil {
			return 0, err
		}
	}

	if c.MockExecRowsAffected == nil {
		return 0, ctx.Err()
	}
	return c.MockExecRowsAffected(ctx, query, args...)
}

// Query implements Connection by recording the query,
// optionally logging it, and then calling MockQuery.
// If MockQuery is nil, it looks up the result in MockQueryResults.
// If no matching result is found, it returns ErrRows wrapping
// sql.ErrNoRows joined with the context error.
func (c *MockConn) Query(ctx context.Context, query string, args ...any) Rows {
	queryFormatter := c.getQueryFormatter()
	queryData, err := NewQueryData(query, args, c.NormalizeQuery)
	if err != nil {
		return NewErrRows(err)
	}

	c.mtx.Lock()
	c.Recordings.Queries = append(c.Recordings.Queries, queryData)
	c.mtx.Unlock()

	if c.QueryLog != nil {
		var err error
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return NewErrRows(err)
			}
		}
		_, err = fmt.Fprint(c.QueryLog, FormatQuery(queryFormatter, query, args...), ";\n")
		if err != nil {
			return NewErrRows(err)
		}
	}

	if c.MockQuery == nil {
		mockRows := c.MockQueryResults[queryData.Format(queryFormatter)]
		if mockRows == nil {
			return NewErrRows(fmt.Errorf("mock %w", sql.ErrNoRows))
		}
		return mockRows
	}
	return c.MockQuery(ctx, query, args...)
}

// Prepare implements Connection by calling MockPrepare.
// If MockPrepare is nil, it returns a MockStmt that delegates
// Exec and Query back to the MockConn.
func (c *MockConn) Prepare(ctx context.Context, query string) (Stmt, error) {
	if c.QueryLog != nil {
		var err error
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return nil, err
			}
		}
		c.mtx.Lock()
		c.StmtNo++
		stmtNo := c.StmtNo
		c.mtx.Unlock()
		_, err = fmt.Fprintf(c.QueryLog, "PREPARE stmt%d AS %s;\n", stmtNo, query)
		if err != nil {
			return nil, err
		}
	}

	if c.MockPrepare == nil {
		stmt := &MockStmt{
			Prepared: query,
			MockExec: func(ctx context.Context, args ...any) error {
				return c.Exec(ctx, query, args...)
			},
			MockExecRowsAffected: func(ctx context.Context, args ...any) (int64, error) {
				return c.ExecRowsAffected(ctx, query, args...)
			},
			MockQuery: func(ctx context.Context, args ...any) Rows {
				return c.Query(ctx, query, args...)
			},
		}
		if c.QueryLog != nil {
			c.mtx.Lock()
			stmtNo := c.StmtNo
			c.mtx.Unlock()
			dealloc := fmt.Sprintf("DEALLOCATE PREPARE stmt%d;\n", stmtNo)
			stmt.MockClose = func() error {
				_, err := fmt.Fprint(c.QueryLog, dealloc)
				return err
			}
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return stmt, nil
	}
	return c.MockPrepare(ctx, query)
}

// DefaultIsolationLevel implements Connection
// by returning sql.LevelDefault.
func (c *MockConn) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

// Transaction implements Connection by calling MockTransaction
// or returning a TransactionState with the current TxID if MockTransaction is nil.
func (c *MockConn) Transaction() TransactionState {
	if c.MockTransaction == nil {
		return TransactionState{
			ID:   c.TxID,
			Opts: nil,
		}
	}
	return c.MockTransaction()
}

// Begin implements Connection by calling MockBegin.
// If MockBegin is nil, it returns a copy of the MockConn
// with TxID set to the given id. Returns an error if id is zero.
func (c *MockConn) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	if c.QueryLog != nil {
		query := "BEGIN"
		if opts != nil {
			if opts.Isolation != sql.LevelDefault {
				query += " ISOLATION LEVEL " + strings.ToUpper(opts.Isolation.String())
			}
			if opts.ReadOnly {
				query += " READ ONLY"
			}
		}
		_, err := fmt.Fprint(c.QueryLog, query+";\n")
		if err != nil {
			return nil, err
		}
	}

	if c.MockBegin == nil {
		tx := c.Clone()
		tx.TxID = id
		return tx, nil
	}
	return c.MockBegin(ctx, id, opts)
}

// Commit implements Connection by calling MockCommit
// or returning nil if MockCommit is nil.
func (c *MockConn) Commit() error {
	if c.QueryLog != nil {
		_, err := fmt.Fprint(c.QueryLog, "COMMIT;\n")
		if err != nil {
			return err
		}
	}

	if c.MockCommit == nil {
		return nil
	}
	return c.MockCommit()
}

// Rollback implements Connection by calling MockRollback
// or returning nil if MockRollback is nil.
func (c *MockConn) Rollback() error {
	if c.QueryLog != nil {
		_, err := fmt.Fprint(c.QueryLog, "ROLLBACK;\n")
		if err != nil {
			return err
		}
	}

	if c.MockRollback == nil {
		return nil
	}
	return c.MockRollback()
}

// ListenOnChannel implements ListenerConnection by registering
// the channel in ListeningOn and calling MockListenOnChannel
// or returning nil if MockListenOnChannel is nil.
func (c *MockConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	c.mtx.Lock()
	if c.ListeningOn == nil {
		c.ListeningOn = make(map[string]struct{})
	}
	c.ListeningOn[channel] = struct{}{}
	c.mtx.Unlock()

	if c.QueryLog != nil {
		_, err := fmt.Fprintf(c.QueryLog, "LISTEN %s;\n", channel)
		if err != nil {
			return err
		}
	}

	if c.MockListenOnChannel == nil {
		return nil
	}
	return c.MockListenOnChannel(channel, onNotify, onUnlisten)
}

// UnlistenChannel implements ListenerConnection by removing
// the channel from ListeningOn and calling MockUnlistenChannel
// or returning nil if MockUnlistenChannel is nil.
func (c *MockConn) UnlistenChannel(channel string) error {
	c.mtx.Lock()
	delete(c.ListeningOn, channel)
	c.mtx.Unlock()

	if c.QueryLog != nil {
		_, err := fmt.Fprintf(c.QueryLog, "UNLISTEN %s;\n", channel)
		if err != nil {
			return err
		}
	}

	if c.MockUnlistenChannel == nil {
		return nil
	}
	return c.MockUnlistenChannel(channel)
}

// IsListeningOnChannel implements ListenerConnection by calling
// MockIsListeningOnChannel or checking the ListeningOn map
// if MockIsListeningOnChannel is nil.
func (c *MockConn) IsListeningOnChannel(channel string) bool {
	if c.MockIsListeningOnChannel == nil {
		c.mtx.Lock()
		_, ok := c.ListeningOn[channel]
		c.mtx.Unlock()
		return ok
	}
	return c.MockIsListeningOnChannel(channel)
}

// Close implements Connection by calling MockClose
// or returning nil if MockClose is nil.
func (c *MockConn) Close() error {
	if c.MockClose == nil {
		return nil
	}
	return c.MockClose()
}
