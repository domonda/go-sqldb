package mockconn

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

func New(queryWriter io.Writer, rowsProvider RowsProvider) sqldb.Connection {
	return &connection{
		queryWriter:      queryWriter,
		listening:        newBoolMap(),
		rowsProvider:     rowsProvider,
		structFieldNamer: sqldb.DefaultStructFieldTagNaming,
	}
}

type connection struct {
	queryWriter      io.Writer
	listening        *boolMap
	rowsProvider     RowsProvider
	structFieldNamer sqldb.StructFieldNamer
}

func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	return &connection{
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

func (conn *connection) Ping(ctx context.Context) error {
	return nil
}

func (conn *connection) Exec(query string, args ...interface{}) error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	return nil
}

func (conn *connection) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	return nil
}

func (conn *connection) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(context.Background(), conn, table, columValues)
}

func (conn *connection) InsertContext(ctx context.Context, table string, columValues sqldb.Values) error {
	return impl.Insert(ctx, conn, table, columValues)
}

func (conn *connection) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return impl.InsertUnique(context.Background(), conn, table, values, onConflict)
}

func (conn *connection) InsertUniqueContext(ctx context.Context, table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return impl.InsertUnique(ctx, conn, table, values, onConflict)
}

func (conn *connection) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(context.Background(), conn, table, values, returning)
}

func (conn *connection) InsertReturningContext(ctx context.Context, table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(ctx, conn, table, values, returning)
}

func (conn *connection) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) InsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) InsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) InsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) Update(table string, values sqldb.Values, where string, args ...interface{}) error {
	return impl.Update(context.Background(), conn, table, values, where, args)
}

func (conn *connection) UpdateContext(ctx context.Context, table string, values sqldb.Values, where string, args ...interface{}) error {
	return impl.Update(ctx, conn, table, values, where, args)
}

func (conn *connection) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowScanner {
	return impl.UpdateReturningRow(context.Background(), conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRowContext(ctx context.Context, table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowScanner {
	return impl.UpdateReturningRow(ctx, conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowsScanner {
	return impl.UpdateReturningRows(context.Background(), conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRowsContext(ctx context.Context, table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowsScanner {
	return impl.UpdateReturningRows(ctx, conn, table, values, returning, where, args)
}

func (conn *connection) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) UpdateStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) UpdateStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) UpdateStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) UpsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) UpsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) UpsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	return conn.QueryRowContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowContext(ctx context.Context, query string, args ...interface{}) sqldb.RowScanner {
	if ctx.Err() != nil {
		return sqldb.RowScannerWithError(ctx.Err())
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
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	if ctx.Err() != nil {
		return sqldb.RowsScannerWithError(ctx.Err())
	}
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	if conn.rowsProvider == nil {
		return sqldb.RowsScannerWithError(nil)
	}
	return conn.rowsProvider.QueryRows(conn.structFieldNamer, query, args...)
}

// IsTransaction returns if the connection is a transaction
func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "BEGIN")
	}
	return transaction{conn}, nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return impl.Transaction(ctx, opts, conn, txFunc)
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
