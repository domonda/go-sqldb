package pqconn

import (
	"context"
	"database/sql"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
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
	return impl.WrapNonNilErrorWithQuery(err, query, argFmt, args)
}

func (conn *transaction) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(conn, table, argFmt, columValues)
}

func (conn *transaction) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return impl.InsertUnique(conn, table, argFmt, values, onConflict)
}

func (conn *transaction) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(conn, table, argFmt, values, returning)
}

func (conn *transaction) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *transaction) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *transaction) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *transaction) InsertUniqueStructIgnoreColumns(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *transaction) Update(table string, values sqldb.Values, where string, args ...interface{}) error {
	return impl.Update(conn, table, values, where, argFmt, args)
}

func (conn *transaction) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowScanner {
	return impl.UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *transaction) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowsScanner {
	return impl.UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *transaction) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *transaction) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *transaction) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *transaction) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return impl.NewRowScanner(rows, conn.structFieldNamer, query, argFmt, args)
}

func (conn *transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	return impl.NewRowsScanner(conn.parent.ctx, rows, conn.structFieldNamer, query, argFmt, args)
}

func (conn *transaction) IsTransaction() bool {
	return true
}

func (conn *transaction) TransactionOptions() (*sql.TxOptions, bool) {
	return conn.opts, true
}

func (conn *transaction) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
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
