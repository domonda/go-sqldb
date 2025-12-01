package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"
)

type QueryRecordings struct {
	Execs   []QueryData
	Queries []QueryData
}

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
// If NormalizeQuery is nil, the query is not normalized.
//
// If TxNo is returned by TransactionState
// and a non zero value simulates a transaction.
type MockConn struct {
	// Configuration
	QueryFormatter QueryFormatter     // StdQueryFormatter{} is used if nil
	NormalizeQuery NormalizeQueryFunc // nil means no normalization
	QueryLog       io.Writer          // nil means no writing of queries

	// Connection state
	TxID        uint64 // Returned by TransactionState
	StmtNo      uint64
	ListeningOn map[string]struct{}

	Recordings       QueryRecordings
	MockQueryResults map[string]*MockRows

	MockConfig               func() *ConnConfig
	MockPing                 func(context.Context, time.Duration) error
	MockStats                func() sql.DBStats
	MockExec                 func(ctx context.Context, query string, args ...any) error
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
}

// MockConn implements ListenerConnection
var _ ListenerConnection = new(MockConn)

func NewMockConn(placeholderPosPrefix string, normalizeQuery NormalizeQueryFunc, queryLog io.Writer) *MockConn {
	return &MockConn{
		QueryFormatter: NewQueryFormatter(placeholderPosPrefix),
		NormalizeQuery: normalizeQuery,
		QueryLog:       queryLog,
	}
}

func (c *MockConn) Clone() *MockConn {
	copy := *c
	copy.ListeningOn = maps.Clone(c.ListeningOn)
	copy.MockQueryResults = maps.Clone(c.MockQueryResults)
	return &copy
}

func (c *MockConn) WithQueryResult(columns []string, rows [][]driver.Value, forQuery string, args ...any) *MockConn {
	queryFormatter := c.QueryFormatter
	if queryFormatter == nil {
		queryFormatter = StdQueryFormatter{}
	}
	normQuery := MustNormalizeAndFormatQuery(c.NormalizeQuery, queryFormatter, forQuery, args...)

	cc := c.Clone()
	if cc.MockQueryResults == nil {
		cc.MockQueryResults = make(map[string]*MockRows)
	}
	cc.MockQueryResults[normQuery] = NewMockRows(columns, rows)
	return cc
}

func (*MockConn) FormatStringLiteral(str string) string {
	return FormatSingleQuoteStringLiteral(str)
}

func (c *MockConn) Config() *ConnConfig {
	if c.MockConfig == nil {
		return &ConnConfig{
			Driver: "MockConn",
		}
	}
	return c.MockConfig()
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

func (c *MockConn) Exec(ctx context.Context, query string, args ...any) (err error) {
	queryFormatter := c.QueryFormatter
	if queryFormatter == nil {
		queryFormatter = StdQueryFormatter{}
	}
	queryData, err := NewQueryData(query, args, c.NormalizeQuery)
	if err != nil {
		return err
	}
	c.Recordings.Execs = append(c.Recordings.Execs, queryData)

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

func (c *MockConn) Query(ctx context.Context, query string, args ...any) Rows {
	queryFormatter := c.QueryFormatter
	if queryFormatter == nil {
		queryFormatter = StdQueryFormatter{}
	}
	queryData, err := NewQueryData(query, args, c.NormalizeQuery)
	if err != nil {
		return NewErrRows(err)
	}
	c.Recordings.Queries = append(c.Recordings.Queries, queryData)

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
			return NewErrRows(ctx.Err())
		}
		return mockRows.WithErr(ctx.Err())
	}
	return c.MockQuery(ctx, query, args...)
}

func (c *MockConn) Prepare(ctx context.Context, query string) (Stmt, error) {
	if c.QueryLog != nil {
		var err error
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return nil, err
			}
		}
		c.StmtNo++
		_, err = fmt.Fprintf(c.QueryLog, "PREPARE stmt%d AS %s;\n", c.StmtNo, query)
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
			MockQuery: func(ctx context.Context, args ...any) Rows {
				return c.Query(ctx, query, args...)
			},
		}
		if c.QueryLog != nil {
			dealloc := fmt.Sprintf("DEALLOCATE PREPARE stmt%d;\n", c.StmtNo)
			stmt.MockClose = func() error {
				_, err := fmt.Fprint(c.QueryLog, dealloc)
				return err
			}
		}
		return stmt, ctx.Err()
	}
	return c.MockPrepare(ctx, query)
}

func (c *MockConn) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

func (c *MockConn) Transaction() TransactionState {
	if c.MockTransaction == nil {
		return TransactionState{
			ID:   c.TxID,
			Opts: nil,
		}
	}
	return c.MockTransaction()
}

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
		tx := *c // copy
		tx.TxID = id
		return &tx, nil
	}
	return c.MockBegin(ctx, id, opts)
}

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

func (c *MockConn) Rollback() error {
	if c.QueryLog != nil {
		_, err := fmt.Fprint(c.QueryLog, "ROLLBACK;\n")
		if err != nil {
			return err
		}
	}

	if c.MockCommit == nil {
		return nil
	}
	return c.MockCommit()
}

func (c *MockConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	if c.ListeningOn == nil {
		c.ListeningOn = make(map[string]struct{})
	}
	c.ListeningOn[channel] = struct{}{}

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

func (c *MockConn) UnlistenChannel(channel string) error {
	delete(c.ListeningOn, channel)

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

func (c *MockConn) IsListeningOnChannel(channel string) bool {
	if c.MockIsListeningOnChannel == nil {
		_, ok := c.ListeningOn[channel]
		return ok
	}
	return c.MockIsListeningOnChannel(channel)
}

func (c *MockConn) Close() error {
	if c.MockClose == nil {
		return nil
	}
	return c.MockClose()
}
