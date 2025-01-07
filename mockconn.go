package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
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
	QueryFormatter // StdQueryFormatter{} is used if nil

	// Connection state
	TxNo              uint64 // Returned by TransactionInfo
	StmtNo            uint64
	ListeningChannels map[string]struct{}
	NormalizeQuery    func(query string) (string, error)

	Output io.Writer

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

func NewPrintMockConn(out io.Writer, queryFormatter QueryFormatter) *MockConn {
	if queryFormatter == nil {
		queryFormatter = StdQueryFormatter{}
	}
	return &MockConn{
		QueryFormatter: queryFormatter,
		Output:         out,
	}
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

func (c *MockConn) Exec(ctx context.Context, query string, args ...any) (err error) {
	if c.Output != nil {
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprint(c.Output, FormatQuery(c.QueryFormatter, query, args...), ";\n")
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
	if c.Output != nil {
		var err error
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return NewErrRows(err)
			}
		}
		_, err = fmt.Fprint(c.Output, FormatQuery(c.QueryFormatter, query, args...), ";\n")
		if err != nil {
			return NewErrRows(err)
		}
	}

	if c.MockQuery == nil {
		return NewErrRows(ctx.Err())
	}
	return c.MockQuery(ctx, query, args...)
}

func (c *MockConn) Prepare(ctx context.Context, query string) (Stmt, error) {
	if c.Output != nil {
		var err error
		if c.NormalizeQuery != nil {
			query, err = c.NormalizeQuery(query)
			if err != nil {
				return nil, err
			}
		}
		c.StmtNo++
		_, err = fmt.Fprintf(c.Output, "PREPARE stmt%d AS %s;\n", c.StmtNo, query)
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
		if c.Output != nil {
			dealloc := fmt.Sprintf("DEALLOCATE PREPARE stmt%d;\n", c.StmtNo)
			stmt.MockClose = func() error {
				_, err := fmt.Fprint(c.Output, dealloc)
				return err
			}
		}
		return stmt, ctx.Err()
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
	if c.Output != nil {
		query := "BEGIN"
		if opts != nil {
			if opts.Isolation != sql.LevelDefault {
				query += " ISOLATION LEVEL " + strings.ToUpper(opts.Isolation.String())
			}
			if opts.ReadOnly {
				query += " READ ONLY"
			}
		}
		_, err := fmt.Fprint(c.Output, query+";\n")
		if err != nil {
			return nil, err
		}
	}

	if c.MockBegin == nil {
		tx := *c // copy
		tx.TxNo = no
		return &tx, nil
	}
	return c.MockBegin(ctx, no, opts)
}

func (c *MockConn) Commit() error {
	if c.Output != nil {
		_, err := fmt.Fprint(c.Output, "COMMIT;\n")
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
	if c.Output != nil {
		_, err := fmt.Fprint(c.Output, "ROLLBACK;\n")
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
	if c.ListeningChannels == nil {
		c.ListeningChannels = make(map[string]struct{})
	}
	c.ListeningChannels[channel] = struct{}{}

	if c.Output != nil {
		_, err := fmt.Fprintf(c.Output, "LISTEN %s;\n", channel)
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
	delete(c.ListeningChannels, channel)

	if c.Output != nil {
		_, err := fmt.Fprintf(c.Output, "UNLISTEN %s;\n", channel)
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
		_, ok := c.ListeningChannels[channel]
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
