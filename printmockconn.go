package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"
)

// printMockConn implements ListenerConnection
var _ ListenerConnection = new(printMockConn)

type printMockConn struct {
	QueryFormatter // StdQueryFormatter{} is used if nil
	out            io.Writer
	listening      map[string]struct{}
	txNo           uint64 // Returned by TransactionInfo
}

func NewPrintMockConn(out io.Writer, queryFormatter QueryFormatter) *printMockConn {
	return &printMockConn{
		QueryFormatter: queryFormatter,
		out:            out,
		listening:      make(map[string]struct{}),
	}
}

func (c *printMockConn) Ping(ctx context.Context, timeout time.Duration) error {
	return nil
}

func (c *printMockConn) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (c *printMockConn) Config() *Config {
	return &Config{Driver: "printMockConn"}
}

func (c *printMockConn) Exec(ctx context.Context, query string, args ...any) error {
	_, err := io.WriteString(c.out, FormatQuery(c.QueryFormatter, query, args...)+";\n")
	return err
}

func (c *printMockConn) Query(ctx context.Context, query string, args ...any) Rows {
	_, err := io.WriteString(c.out, FormatQuery(c.QueryFormatter, query, args...)+";\n")
	return NewErrRows(err) // Empty result
}

func (c *printMockConn) Prepare(ctx context.Context, query string) (Stmt, error) {
	_, err := fmt.Fprintf(c.out, "PREPARE %s;\n", query)
	if err != nil {
		return nil, err
	}
	return NewUnpreparedStmt(c, query), nil
}

func (c *printMockConn) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return c.txNo, nil
}

func (c *printMockConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	optsString := ""
	if opts != nil {
		if opts.Isolation != sql.LevelDefault {
			optsString = " ISOLATION LEVEL " + opts.Isolation.String()
		}
		if opts.ReadOnly {
			optsString += " READ ONLY"
		}
	}
	_, err := fmt.Fprintf(c.out, "BEGIN%s;\n", optsString)
	if err != nil {
		return nil, err
	}
	tx := *c // copy
	tx.txNo = no
	return &tx, nil
}

func (c *printMockConn) Commit() error {
	_, err := io.WriteString(c.out, "COMMIT;\n")
	return err
}

func (c *printMockConn) Rollback() error {
	_, err := io.WriteString(c.out, "ROLLBACK;\n")
	return err
}

func (c *printMockConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	_, err := fmt.Fprintf(c.out, "LISTEN %s;\n", channel)
	if err != nil {
		return err
	}
	c.listening[channel] = struct{}{}
	return nil
}

func (c *printMockConn) UnlistenChannel(channel string) error {
	_, err := fmt.Fprintf(c.out, "UNLISTEN %s;\n", channel)
	if err != nil {
		return err
	}
	delete(c.listening, channel)
	return nil
}

func (c *printMockConn) IsListeningOnChannel(channel string) bool {
	_, ok := c.listening[channel]
	return ok
}

func (c *printMockConn) Close() error {
	return nil
}
