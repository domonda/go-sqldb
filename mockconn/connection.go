package mockconn

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

var DefaultArgFmt = "$%d"

func New(ctx context.Context, queryWriter io.Writer, rowsProvider RowsProvider) sqldb.Connection {
	return &connection{
		ctx:              ctx,
		queryWriter:      queryWriter,
		listening:        newBoolMap(),
		rowsProvider:     rowsProvider,
		structFieldNamer: sqldb.DefaultStructFieldMapping,
		argFmt:           DefaultArgFmt,
	}
}

type connection struct {
	ctx              context.Context
	queryWriter      io.Writer
	listening        *boolMap
	rowsProvider     RowsProvider
	structFieldNamer sqldb.StructFieldMapper
	argFmt           string
}

func (conn *connection) Context() context.Context { return conn.ctx }

func (conn *connection) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == conn.ctx {
		return conn
	}
	return &connection{
		ctx:              ctx,
		queryWriter:      conn.queryWriter,
		listening:        conn.listening,
		rowsProvider:     conn.rowsProvider,
		structFieldNamer: conn.structFieldNamer,
		argFmt:           conn.argFmt,
	}
}

func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldMapper) sqldb.Connection {
	return &connection{
		ctx:              conn.ctx,
		queryWriter:      conn.queryWriter,
		listening:        conn.listening,
		rowsProvider:     conn.rowsProvider,
		structFieldNamer: namer,
		argFmt:           conn.argFmt,
	}
}

func (conn *connection) StructFieldNamer() sqldb.StructFieldMapper {
	return conn.structFieldNamer
}

func (conn *connection) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (conn *connection) Config() *sqldb.Config {
	return &sqldb.Config{Driver: "mockconn", Host: "localhost", Database: "mock"}
}

func (conn *connection) Ping(time.Duration) error {
	return nil
}

func (conn *connection) Now() (time.Time, error) {
	return time.Now(), nil
}

func (conn *connection) Exec(query string, args ...any) error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	return nil
}

func (conn *connection) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(conn, table, conn.argFmt, columValues)
}

func (conn *connection) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return impl.InsertUnique(conn, table, conn.argFmt, values, onConflict)
}

func (conn *connection) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(conn, table, conn.argFmt, values, returning)
}

func (conn *connection) InsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) InsertUniqueStruct(table string, rowStruct any, onConflict string, ignoreColumns ...sqldb.ColumnFilter) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) Update(table string, values sqldb.Values, where string, args ...any) error {
	return impl.Update(conn, table, values, where, conn.argFmt, args)
}

func (conn *connection) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	return impl.UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	return impl.UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
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

func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
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
