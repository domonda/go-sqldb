package mockconn

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

var DefaultArgFmt = "$%d"

func New(ctx context.Context, queryWriter io.Writer, rowsProvider RowsProvider) sqldb.Connection {
	return &connection{
		ctx:               ctx,
		queryWriter:       queryWriter,
		listening:         newBoolMap(),
		rowsProvider:      rowsProvider,
		structFieldMapper: sqldb.DefaultStructFieldMapping,
		argFmt:            DefaultArgFmt,
	}
}

type connection struct {
	ctx               context.Context
	queryWriter       io.Writer
	listening         *boolMap
	rowsProvider      RowsProvider
	structFieldMapper sqldb.StructFieldMapper
	converter         driver.ValueConverter
	argFmt            string
}

func (conn *connection) clone() *connection {
	c := *conn
	return &c
}

func (conn *connection) Context() context.Context { return conn.ctx }

func (conn *connection) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == conn.ctx {
		return conn
	}
	c := conn.clone()
	c.ctx = ctx
	return c
}

func (conn *connection) WithStructFieldMapper(mapper sqldb.StructFieldMapper) sqldb.Connection {
	c := conn.clone()
	c.structFieldMapper = mapper
	return c
}

func (conn *connection) StructFieldMapper() sqldb.StructFieldMapper {
	return conn.structFieldMapper
}

func (conn *connection) ValidateColumnName(name string) error {
	return validateColumnName(name)
}

func (conn *connection) Ping(time.Duration) error {
	return nil
}

func (conn *connection) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (conn *connection) Config() *sqldb.Config {
	return &sqldb.Config{Driver: "mockconn", Host: "localhost", Database: "mock"}
}

func (conn *connection) Now() (time.Time, error) {
	return time.Now(), nil
}

func (conn *connection) Exec(query string, args ...any) error {
	return impl.Exec(conn.ctx, conn, query, args, conn.converter, conn.argFmt)
}

func (conn *connection) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	return nil, nil
}

func (conn *connection) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(conn.ctx, conn, table, columValues, conn.converter, conn.argFmt)
}

func (conn *connection) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return impl.InsertUnique(conn.ctx, conn, table, values, onConflict, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *connection) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(conn.ctx, conn, table, values, returning, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *connection) InsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.InsertStruct(conn.ctx, conn, table, rowStruct, conn.structFieldMapper, ignoreColumns, conn.converter, conn.argFmt)
}

func (conn *connection) InsertStructs(table string, rowStructs any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.InsertStructs(conn, table, rowStructs, ignoreColumns...)
}

func (conn *connection) InsertUniqueStruct(table string, rowStruct any, onConflict string, ignoreColumns ...sqldb.ColumnFilter) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn.ctx, conn, conn.structFieldMapper, table, rowStruct, onConflict, ignoreColumns, conn.converter, conn.argFmt)
}

func (conn *connection) Update(table string, values sqldb.Values, where string, args ...any) error {
	return impl.Update(conn.ctx, conn, table, values, where, args, conn.converter, conn.argFmt)
}

func (conn *connection) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	return impl.UpdateReturningRow(conn.ctx, conn, table, values, returning, where, args, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *connection) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	return impl.UpdateReturningRows(conn.ctx, conn, table, values, returning, where, args, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *connection) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpdateStruct(conn.ctx, conn, table, rowStruct, conn.structFieldMapper, ignoreColumns, conn.converter, conn.argFmt)
}

func (conn *connection) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpsertStruct(conn.ctx, conn, table, rowStruct, conn.structFieldMapper, conn.argFmt, ignoreColumns)
}

func (conn *connection) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	panic("todo")
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
	return conn.rowsProvider.QueryRow(conn.structFieldMapper, query, args...)
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
	return conn.rowsProvider.QueryRows(conn.structFieldMapper, query, args...)
}

func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) TransactionNo() uint64 {
	return 0
}

func (conn *connection) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (conn *connection) Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error) {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "BEGIN")
	}
	return transaction{conn, opts, no}, nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) ListenChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
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
