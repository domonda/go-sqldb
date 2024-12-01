package impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
)

// Connection returns a generic sqldb.Connection implementation
// for an existing sql.DB connection.
// argFmt is the format string for argument placeholders like "?" or "$%d"
// that will be replaced error messages to format a complete query.
func Connection(ctx context.Context, db *sql.DB, config *sqldb.Config, validateColumnName func(string) error, argFmt string) sqldb.Connection {
	return &connection{
		ctx:                ctx,
		db:                 db,
		config:             config,
		structFieldNamer:   sqldb.DefaultStructFieldMapping,
		argFmt:             argFmt,
		validateColumnName: validateColumnName,
	}
}

type connection struct {
	ctx                context.Context
	db                 *sql.DB
	config             *sqldb.Config
	structFieldNamer   sqldb.StructFieldMapper
	argFmt             string
	validateColumnName func(string) error
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

func (conn *connection) WithStructFieldMapper(namer sqldb.StructFieldMapper) sqldb.Connection {
	c := conn.clone()
	c.structFieldNamer = namer
	return c
}

func (conn *connection) StructFieldMapper() sqldb.StructFieldMapper {
	return conn.structFieldNamer
}

func (conn *connection) Ping(timeout time.Duration) error {
	ctx := conn.ctx
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.db.PingContext(ctx)
}

func (conn *connection) Stats() sql.DBStats {
	return conn.db.Stats()
}

func (conn *connection) Config() *sqldb.Config {
	return conn.config
}

func (conn *connection) ValidateColumnName(name string) error {
	return conn.validateColumnName(name)
}

func (conn *connection) Exec(query string, args ...any) error {
	_, err := conn.db.ExecContext(conn.ctx, query, args...)
	return WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
}

func (conn *connection) Insert(table string, columValues sqldb.Values) error {
	return Insert(conn, table, conn.argFmt, columValues)
}

func (conn *connection) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return InsertUnique(conn, table, conn.argFmt, values, onConflict)
}

func (conn *connection) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return InsertReturning(conn, table, conn.argFmt, values, returning)
}

func (conn *connection) InsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return InsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) InsertStructs(table string, rowStructs any, ignoreColumns ...sqldb.ColumnFilter) error {
	return InsertStructs(conn, table, rowStructs, ignoreColumns...)
}

func (conn *connection) InsertUniqueStruct(table string, rowStruct any, onConflict string, ignoreColumns ...sqldb.ColumnFilter) (inserted bool, err error) {
	return InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) Update(table string, values sqldb.Values, where string, args ...any) error {
	return Update(conn, table, values, where, conn.argFmt, args)
}

func (conn *connection) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	return UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	return UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, conn.argFmt, ignoreColumns)
}

func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return NewRowScanner(rows, conn.structFieldNamer, query, conn.argFmt, args)
}

func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	return NewRowsScanner(conn.ctx, rows, conn.structFieldNamer, query, conn.argFmt, args)
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
	tx, err := conn.db.BeginTx(conn.ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn, tx, opts, no), nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return false
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
