package impl

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
	parent           *connection
	tx               *sql.Tx
	opts             *sql.TxOptions
	structFieldNamer sqldb.StructFieldNamer
}

func newTransaction(parent *connection, tx *sql.Tx, opts *sql.TxOptions) *transaction {
	return &transaction{
		parent:           parent,
		tx:               tx,
		opts:             opts,
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
	return newTransaction(parent, conn.tx, conn.opts)
}

func (conn *transaction) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	c := conn.clone()
	c.structFieldNamer = namer
	return c
}

func (conn *transaction) StructFieldNamer() sqldb.StructFieldNamer {
	return conn.structFieldNamer
}

func (conn *transaction) Ping(timeout time.Duration) error { return conn.parent.Ping(timeout) }
func (conn *transaction) Stats() sql.DBStats               { return conn.parent.Stats() }
func (conn *transaction) Config() *sqldb.Config            { return conn.parent.Config() }

func (conn *transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return WrapNonNilErrorWithQuery(err, query, conn.parent.argFmt, args)
}

func (conn *transaction) Insert(table string, columValues sqldb.Values) error {
	return Insert(conn, table, conn.parent.argFmt, columValues)
}

func (conn *transaction) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return InsertUnique(conn, table, conn.parent.argFmt, values, onConflict)
}

func (conn *transaction) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return InsertReturning(conn, table, conn.parent.argFmt, values, returning)
}

func (conn *transaction) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return InsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.parent.argFmt, nil, restrictToColumns)
}

func (conn *transaction) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return InsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.parent.argFmt, ignoreColumns, nil)
}

func (conn *transaction) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, conn.parent.argFmt, nil, restrictToColumns)
}

func (conn *transaction) InsertUniqueStructIgnoreColumns(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, conn.parent.argFmt, ignoreColumns, nil)
}

func (conn *transaction) Update(table string, values sqldb.Values, where string, args ...interface{}) error {
	return Update(conn, table, values, where, conn.parent.argFmt, args)
}

func (conn *transaction) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowScanner {
	return UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *transaction) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowsScanner {
	return UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *transaction) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, conn.parent.argFmt, nil, restrictToColumns)
}

func (conn *transaction) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, conn.parent.argFmt, ignoreColumns, nil)
}

func (conn *transaction) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.parent.argFmt, nil, restrictToColumns)
}

func (conn *transaction) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.parent.argFmt, ignoreColumns, nil)
}

func (conn *transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, conn.parent.argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return NewRowScanner(rows, conn.structFieldNamer, query, conn.parent.argFmt, args)
}

func (conn *transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
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
