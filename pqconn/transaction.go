package pqconn

import (
	"context"
	"database/sql"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
)

type transaction struct {
	// The parent non-transaction connection is needed
	// for its ctx, Ping(), Stats(), and Config()
	parent            *connection
	tx                *sql.Tx
	opts              *sql.TxOptions
	structFieldMapper reflection.StructFieldMapper
}

func newTransaction(parent *connection, tx *sql.Tx, opts *sql.TxOptions) *transaction {
	return &transaction{
		parent:            parent,
		tx:                tx,
		opts:              opts,
		structFieldMapper: parent.structFieldMapper,
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

func (conn *transaction) WithStructFieldMapper(mapper reflection.StructFieldMapper) sqldb.Connection {
	c := conn.clone()
	c.structFieldMapper = mapper
	return c
}

func (conn *transaction) StructFieldMapper() reflection.StructFieldMapper {
	return conn.structFieldMapper
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
	return conn.parent.config.Err
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
	return sqldb.WrapNonNilErrorWithQuery(err, query, conn, args)
}

func (conn *transaction) QueryRow(query string, args ...any) sqldb.RowScanner {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = sqldb.WrapNonNilErrorWithQuery(err, query, conn, args)
		return sqldb.RowScannerWithError(err)
	}
	return sqldb.NewRowScanner(rows, conn.structFieldMapper, query, conn, args)
}

func (conn *transaction) QueryRows(query string, args ...any) sqldb.RowsScanner {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = sqldb.WrapNonNilErrorWithQuery(err, query, conn, args)
		return sqldb.RowsScannerWithError(err)
	}
	return sqldb.NewRowsScanner(conn.parent.ctx, rows, conn.structFieldMapper, query, conn, args)
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
