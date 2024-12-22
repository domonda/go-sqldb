package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type genericTxConn struct {
	QueryFormatter
	// The parent non-transaction connection is needed
	// for Ping(), Stats(), Config(), and Begin().
	parent *genericConn
	tx     *sql.Tx
	opts   *sql.TxOptions
	no     uint64
}

func newGenericTxConn(parent *genericConn, tx *sql.Tx, opts *sql.TxOptions, no uint64) *genericTxConn {
	return &genericTxConn{
		QueryFormatter: parent.QueryFormatter,
		parent:         parent,
		tx:             tx,
		opts:           opts,
		no:             no,
	}
}

func (conn *genericTxConn) Ping(ctx context.Context, timeout time.Duration) error {
	return conn.parent.Ping(ctx, timeout)
}
func (conn *genericTxConn) Stats() sql.DBStats { return conn.parent.Stats() }
func (conn *genericTxConn) Config() *Config    { return conn.parent.Config() }

func (conn *genericTxConn) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.tx.ExecContext(ctx, query, args...)
	return err
}

func (conn *genericTxConn) Query(ctx context.Context, query string, args ...any) Rows {
	rows, err := conn.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return NewErrRows(err)
	}
	return rows
}

func (conn *genericTxConn) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return conn.no, conn.opts
}

func (conn *genericTxConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	tx, err := conn.parent.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newGenericTxConn(conn.parent, tx, opts, no), nil
}

func (conn *genericTxConn) Commit() error {
	return conn.tx.Commit()
}

func (conn *genericTxConn) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *genericTxConn) Close() error {
	return conn.Rollback()
}
