package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// NewGenericConn returns a generic Connection implementation
// for an existing sql.DB connection.
func NewGenericConn(db *sql.DB, config *Config, queryFormatter QueryFormatter) Connection {
	return &genericConn{
		QueryFormatter: queryFormatter,
		db:             db,
		config:         config,
	}
}

type genericConn struct {
	QueryFormatter
	db     *sql.DB
	config *Config
}

func (conn *genericConn) Ping(ctx context.Context, timeout time.Duration) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.db.PingContext(ctx)
}

func (conn *genericConn) Stats() sql.DBStats {
	return conn.db.Stats()
}

func (conn *genericConn) Config() *Config {
	return conn.config
}

func (conn *genericConn) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.db.ExecContext(ctx, query, args...)
	return err
}

func (conn *genericConn) Query(ctx context.Context, query string, args ...any) Rows {
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		return NewErrRows(err)
	}
	return rows
}

func (conn *genericConn) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return 0, nil
}

func (conn *genericConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	tx, err := conn.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newGenericTxConn(conn, tx, opts, no), nil
}

func (conn *genericConn) Commit() error {
	return ErrNotWithinTransaction
}

func (conn *genericConn) Rollback() error {
	return ErrNotWithinTransaction
}

func (conn *genericConn) Close() error {
	return conn.db.Close()
}
