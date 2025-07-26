package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type genericTx struct {
	QueryFormatter
	// The parent non-transaction connection is needed
	// for Ping(), Stats(), Config(), and Begin().
	parent *genericConn
	tx     *sql.Tx
	opts   *sql.TxOptions
	no     uint64
}

func newGenericTx(parent *genericConn, tx *sql.Tx, opts *sql.TxOptions, no uint64) *genericTx {
	return &genericTx{
		QueryFormatter: parent.QueryFormatter,
		parent:         parent,
		tx:             tx,
		opts:           opts,
		no:             no,
	}
}

func (conn *genericTx) Ping(ctx context.Context, timeout time.Duration) error {
	return conn.parent.Ping(ctx, timeout)
}
func (conn *genericTx) Stats() sql.DBStats { return conn.parent.Stats() }
func (conn *genericTx) Config() *Config    { return conn.parent.Config() }

func (conn *genericTx) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.tx.ExecContext(ctx, query, args...)
	return err
}

func (conn *genericTx) Query(ctx context.Context, query string, args ...any) Rows {
	rows, err := conn.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return NewErrRows(err)
	}
	return rows
}

func (conn *genericTx) Prepare(ctx context.Context, query string) (Stmt, error) {
	stmt, err := conn.tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return NewStmt(stmt, query), nil
}

func (conn *genericTx) TransactionInfo() TransactionInfo {
	return TransactionInfo{
		No:                    conn.no,
		Opts:                  conn.opts,
		DefaultIsolationLevel: conn.parent.defaultIsolationLevel,
	}
}

func (conn *genericTx) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	tx, err := conn.parent.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newGenericTx(conn.parent, tx, opts, no), nil
}

func (conn *genericTx) Commit() error {
	return conn.tx.Commit()
}

func (conn *genericTx) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *genericTx) Close() error {
	return conn.Rollback()
}
