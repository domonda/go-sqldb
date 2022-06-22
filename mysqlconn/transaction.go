package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
)

type transaction struct {
	// The parent non-transaction connection is needed
	// for its ctx, Ping(), Stats(), and Config()
	parent *connection
	tx     *sql.Tx
	opts   *sql.TxOptions
}

func newTransaction(parent *connection, tx *sql.Tx, opts *sql.TxOptions) *transaction {
	return &transaction{
		parent: parent,
		tx:     tx,
		opts:   opts,
	}
}

func (conn *transaction) clone() *transaction {
	c := *conn
	return &c
}

func (conn *transaction) Context() context.Context { return conn.parent.ctx }

func (conn *transaction) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == conn.parent.ctx {
		return conn
	}
	parent := conn.parent.clone()
	parent.ctx = ctx
	return newTransaction(parent, conn.tx, conn.opts)
}

func (conn *transaction) Ping(timeout time.Duration) error { return conn.parent.Ping(timeout) }
func (conn *transaction) Stats() sql.DBStats               { return conn.parent.Stats() }
func (conn *transaction) Config() *sqldb.Config            { return conn.parent.Config() }

func (conn *transaction) ValidateColumnName(name string) error {
	return validateColumnName(name)
}

func (conn *transaction) ParamPlaceholder(index int) string {
	return conn.parent.ParamPlaceholder(index)
}

func (conn *transaction) Err() error {
	return conn.parent.Err()
}

func (conn *transaction) Now() (now time.Time, err error) {
	err = conn.QueryRow(`select now()`).Scan(&now)
	if err != nil {
		return time.Time{}, err
	}
	return now, nil
}

func (conn *transaction) Exec(query string, args ...any) error {
	_, err := conn.tx.Exec(query, args...)
	return sqldb.WrapErrorWithQuery(err, query, conn, args)
}

func (conn *transaction) QueryRow(query string, args ...any) sqldb.Row {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = sqldb.WrapErrorWithQuery(err, query, conn, args)
		return sqldb.RowWithError(err)
	}
	return sqldb.NewRow(conn.parent.ctx, rows, query, conn, args)
}

func (conn *transaction) QueryRows(query string, args ...any) sqldb.Rows {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = sqldb.WrapErrorWithQuery(err, query, conn, args)
		return sqldb.RowsWithError(err)
	}
	return sqldb.NewRows(conn.parent.ctx, rows, query, conn, args)
}

func (conn *transaction) IsTransaction() bool {
	return true
}

func (conn *transaction) TransactionOptions() (*sql.TxOptions, bool) {
	return conn.opts, true
}

func (conn *transaction) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	tx, err := conn.parent.db.BeginTx(conn.parent.ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn.parent, tx, opts), nil
}

func (conn *transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn *transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return fmt.Errorf("notifications %w", sqldb.ErrNotSupported)
}

func (conn *transaction) UnlistenChannel(channel string) (err error) {
	return fmt.Errorf("notifications %w", sqldb.ErrNotSupported)
}

func (conn *transaction) IsListeningOnChannel(channel string) bool {
	return false
}

func (conn *transaction) Close() error {
	return conn.Rollback()
}
