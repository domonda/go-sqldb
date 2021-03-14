package mockconn

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

func New(ctx context.Context, queryWriter io.Writer, rowsProvider RowsProvider) sqldb.Connection {
	return &connection{
		ctx:              ctx,
		queryWriter:      queryWriter,
		listening:        newBoolMap(),
		rowsProvider:     rowsProvider,
		structFieldNamer: sqldb.DefaultStructFieldTagNaming,
	}
}

type connection struct {
	ctx              context.Context
	queryWriter      io.Writer
	listening        *boolMap
	rowsProvider     RowsProvider
	structFieldNamer sqldb.StructFieldNamer
}

func (conn *connection) WithContext(ctx context.Context) sqldb.Connection {
	return &connection{
		ctx:              ctx,
		queryWriter:      conn.queryWriter,
		listening:        conn.listening,
		rowsProvider:     conn.rowsProvider,
		structFieldNamer: conn.structFieldNamer,
	}
}

func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	return &connection{
		ctx:              conn.ctx,
		queryWriter:      conn.queryWriter,
		listening:        conn.listening,
		rowsProvider:     conn.rowsProvider,
		structFieldNamer: namer,
	}
}

func (conn *connection) StructFieldNamer() sqldb.StructFieldNamer {
	return conn.structFieldNamer
}

func (conn *connection) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (conn *connection) Config() *sqldb.Config {
	return nil
}

func (conn *connection) Ping() error {
	return nil
}

func (conn *connection) Exec(query string, args ...interface{}) error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	return nil
}

func (conn *connection) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(conn, table, columValues)
}

func (conn *connection) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return impl.InsertUnique(conn, table, values, onConflict)
}

func (conn *connection) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(conn, table, values, returning)
}

func (conn *connection) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) InsertUniqueStructIgnoreColumns(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) Update(table string, values sqldb.Values, where string, args ...interface{}) error {
	return impl.Update(conn, table, values, where, args)
}

func (conn *connection) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowScanner {
	return impl.UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowsScanner {
	return impl.UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	if conn.ctx.Err() != nil {
		return sqldb.RowScannerWithError(conn.ctx.Err())
	}
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	if conn.rowsProvider == nil {
		return sqldb.RowScannerWithError(nil)
	}
	return conn.rowsProvider.QueryRow(conn.structFieldNamer, query, args...)
}

func (conn *connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	if conn.ctx.Err() != nil {
		return sqldb.RowsScannerWithError(conn.ctx.Err())
	}
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	if conn.rowsProvider == nil {
		return sqldb.RowsScannerWithError(nil)
	}
	return conn.rowsProvider.QueryRows(conn.structFieldNamer, query, args...)
}

func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (conn *connection) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "BEGIN")
	}
	return transaction{conn, opts}, nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	conn.listening.Set(channel, true)
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "LISTEN "+channel)
	}
	return nil
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	conn.listening.Set(channel, false)
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "UNLISTEN "+channel)
	}
	return nil
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return conn.listening.Get(channel)
}

func (conn *connection) Close() error {
	return nil
}
