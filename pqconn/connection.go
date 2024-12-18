package pqconn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

const (
	Driver = "postgres"

	argFmt = "$%d"
)

// New creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/lib/pq as driver implementation.
// The connection is pinged with the passed context
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}
	config.DefaultIsolationLevel = sql.LevelReadCommitted // postgres default

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &connection{
		ctx:              ctx,
		db:               db,
		config:           config,
		structFieldNamer: sqldb.DefaultStructFieldMapping,
	}, nil
}

// MustNew creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/lib/pq as driver implementation.
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
	structFieldNamer sqldb.StructFieldMapper
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

func (conn *connection) Placeholder(paramIndex int) string {
	return fmt.Sprintf(argFmt, paramIndex+1)
}

func (conn *connection) ValidateColumnName(name string) error {
	return validateColumnName(name)
}

func (conn *connection) Exec(query string, args ...any) error {
	impl.WrapArrayArgs(args)
	_, err := conn.db.ExecContext(conn.ctx, query, args...)
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (conn *connection) Update(table string, values sqldb.Values, where string, args ...any) error {
	return impl.Update(conn, table, values, where, argFmt, args)
}

func (conn *connection) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	return impl.UpdateReturningRow(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	return impl.UpdateReturningRows(conn, table, values, returning, where, args)
}

func (conn *connection) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns)
}

func (conn *connection) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, argFmt, ignoreColumns)
}

func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
	impl.WrapArrayArgs(args)
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = wrapKnownErrors(err)
		return sqldb.RowScannerWithError(err)
	}
	return impl.NewRowScanner(rows, conn.structFieldNamer, query, argFmt, args)
}

func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
	impl.WrapArrayArgs(args)
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = wrapKnownErrors(err)
		return sqldb.RowsScannerWithError(err)
	}
	return impl.NewRowsScanner(conn.ctx, rows, conn.structFieldNamer, query, argFmt, args)
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
	return conn.getOrCreateListener().listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return conn.getListenerOrNil().unlistenChannel(channel)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return conn.getListenerOrNil().isListeningOnChannel(channel)
}

func (conn *connection) Close() error {
	conn.getListenerOrNil().close()
	return conn.db.Close()
}
