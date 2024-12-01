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

func (conn *transaction) ValidateColumnName(name string) error {
	return validateColumnName(name)
}

func (conn *transaction) Exec(query string, args ...any) error {
	impl.WrapArrayArgs(args)
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

func (conn *transaction) InsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns)
}

func (conn *transaction) InsertUniqueStruct(table string, rowStruct any, onConflict string, ignoreColumns ...sqldb.ColumnFilter) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, argFmt, ignoreColumns)
}

func (conn *transaction) Update(table string, values sqldb.Values, where string, args ...any) error {
	return impl.Update(conn, table, values, where, argFmt, args)
}

func (conn *transaction) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	return impl.UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *transaction) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	return impl.UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *transaction) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns)
}

func (conn *transaction) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns)
}

func (conn *transaction) InsertStructs(table string, rowStructs any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.InsertStructs(conn, table, rowStructs, ignoreColumns...)
}

func (conn *transaction) QueryRow(query string, args ...any) sqldb.RowScanner {
	impl.WrapArrayArgs(args)
	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return impl.NewRowScanner(rows, conn.structFieldNamer, query, argFmt, args)
}

func (conn *transaction) QueryRows(query string, args ...any) sqldb.RowsScanner {
	impl.WrapArrayArgs(args)
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
