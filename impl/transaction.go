package impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
)

type transaction struct {
	// The parent non-transaction connection is needed
	// for its ctx, Ping(), Stats(), and Config()
	parent           *connection
	tx               *sql.Tx
	opts             *sql.TxOptions
	no               uint64
	structFieldNamer sqldb.StructFieldMapper
}

func newTransaction(parent *connection, tx *sql.Tx, opts *sql.TxOptions, no uint64) *transaction {
	return &transaction{
		parent:           parent,
		tx:               tx,
		opts:             opts,
		no:               no,
		structFieldNamer: parent.structFieldNamer,
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
	return newTransaction(parent, conn.tx, conn.opts, conn.no)
}

func (conn *transaction) WithStructFieldMapper(namer sqldb.StructFieldMapper) sqldb.Connection {
	c := conn.clone()
	c.structFieldNamer = namer
	return c
}

func (conn *transaction) StructFieldMapper() sqldb.StructFieldMapper {
	return conn.structFieldNamer
}

func (conn *transaction) Ping(timeout time.Duration) error { return conn.parent.Ping(timeout) }
func (conn *transaction) Stats() sql.DBStats               { return conn.parent.Stats() }
func (conn *transaction) Config() *sqldb.Config            { return conn.parent.Config() }
func (conn *transaction) Placeholder(paramIndex int) string {
	return conn.parent.Placeholder(paramIndex)
}

func (conn *transaction) ValidateColumnName(name string) error {
	return conn.parent.validateColumnName(name)
}

func (conn *transaction) Exec(query string, args ...any) error {
	_, err := conn.tx.Exec(query, args...)
	return WrapNonNilErrorWithQuery(err, query, conn.parent.argFmt, args)
}

func (conn *transaction) QueryRow(query string, args ...any) sqldb.RowScanner {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, conn.parent.argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return NewRowScanner(rows, conn.structFieldNamer, query, conn.parent.argFmt, args)
}

func (conn *transaction) QueryRows(query string, args ...any) sqldb.RowsScanner {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, conn.parent.argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	return NewRowsScanner(conn.parent.ctx, rows, conn.structFieldNamer, query, conn.parent.argFmt, args)
}

func (conn *transaction) IsTransaction() bool {
	return true
}

func (conn *transaction) TransactionNo() uint64 {
	return conn.no
}

func (conn *transaction) TransactionOptions() (*sql.TxOptions, bool) {
	return conn.opts, true
}

func (conn *transaction) Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error) {
	tx, err := conn.parent.db.BeginTx(conn.parent.ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn.parent, tx, opts, no), nil
}

func (conn *transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn *transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (conn *transaction) UnlistenChannel(channel string) (err error) {
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (conn *transaction) IsListeningOnChannel(channel string) bool {
	return false
}

func (conn *transaction) Close() error {
	return conn.Rollback()
}
