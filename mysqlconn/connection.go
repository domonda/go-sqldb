package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // Use driver

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

const argFmt = "?"

// New creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != "mysql" {
		return nil, fmt.Errorf(`invalid driver %q, mysqlconn expects "mysql"`, config.Driver)
	}
	config.DefaultIsolationLevel = sql.LevelRepeatableRead // mysql default

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &connection{
		ctx:              ctx,
		db:               db,
		config:           config,
		structFieldNamer: sqldb.DefaultStructFieldTagNaming,
	}, nil
}

// MustNew creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
// Errors are paniced.
func MustNew(ctx context.Context, config *sqldb.Config) sqldb.Connection {
	conn, err := New(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	ctx              context.Context
	db               *sql.DB
	config           *sqldb.Config
	structFieldNamer sqldb.StructFieldNamer
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

func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	c := conn.clone()
	c.structFieldNamer = namer
	return c
}

func (conn *connection) StructFieldNamer() sqldb.StructFieldNamer {
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

func (conn *connection) Exec(query string, args ...interface{}) error {
	_, err := conn.db.ExecContext(conn.ctx, query, args...)
	return impl.WrapNonNilErrorWithQuery(err, query, argFmt, args)
}

func (conn *connection) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(conn, table, argFmt, columValues)
}

func (conn *connection) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return impl.InsertUnique(conn, table, argFmt, values, onConflict)
}

func (conn *connection) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(conn, table, argFmt, values, returning)
}

func (conn *connection) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *connection) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *connection) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *connection) InsertUniqueStructIgnoreColumns(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(conn, table, rowStruct, onConflict, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *connection) Update(table string, values sqldb.Values, where string, args ...interface{}) error {
	return impl.Update(conn, table, values, where, argFmt, args)
}

func (conn *connection) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowScanner {
	return impl.UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowsScanner {
	return impl.UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *connection) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *connection) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, nil, restrictToColumns)
}

func (conn *connection) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns, nil)
}

func (conn *connection) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return impl.NewRowScanner(rows, conn.structFieldNamer, query, argFmt, args)
}

func (conn *connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	return impl.NewRowsScanner(conn.ctx, rows, conn.structFieldNamer, query, argFmt, args)
}

func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (conn *connection) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	tx, err := conn.db.BeginTx(conn.ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn, tx, opts), nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return fmt.Errorf("notifications %w", sqldb.ErrNotSupported)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return fmt.Errorf("notifications %w", sqldb.ErrNotSupported)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return false
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
