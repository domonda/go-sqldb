package pqconn

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/domonda/go-sqldb"
)

type transaction struct {
	QueryFormatter

	// The parent non-transaction connection is needed
	// for its ctx, Ping(), Stats(), and Config()
	parent *connection
	tx     *sql.Tx
	opts   *sql.TxOptions
	id     uint64
}

func newTransaction(parent *connection, tx *sql.Tx, opts *sql.TxOptions, id uint64) *transaction {
	return &transaction{
		parent: parent,
		tx:     tx,
		opts:   opts,
		id:     id,
	}
}

func (conn *transaction) Ping(ctx context.Context, timeout time.Duration) error {
	return conn.parent.Ping(ctx, timeout)
}
func (conn *transaction) Stats() sql.DBStats    { return conn.parent.Stats() }
func (conn *transaction) Config() *sqldb.Config { return conn.parent.Config() }
func (conn *transaction) FormatPlaceholder(paramIndex int) string {
	return conn.parent.FormatPlaceholder(paramIndex)
}

func (conn *transaction) Exec(ctx context.Context, query string, args ...any) error {
	wrapArrayArgs(args)
	_, err := conn.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (conn *transaction) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	wrapArrayArgs(args)
	rows, err := conn.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	return rows
}

func (conn *transaction) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	s, err := conn.tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return stmt{query, s}, nil
}

func (*transaction) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelReadCommitted // postgres default
}

func (conn *transaction) Transaction() sqldb.TransactionState {
	return sqldb.TransactionState{
		ID:   conn.id,
		Opts: conn.opts,
	}
}

func (conn *transaction) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	tx, err := conn.parent.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn.parent, tx, opts, id), nil
}

func (conn *transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn *transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn *transaction) UnlistenChannel(channel string) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn *transaction) IsListeningOnChannel(channel string) bool {
	return false
}

func (conn *transaction) Close() error {
	return conn.Rollback()
}
