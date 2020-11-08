package pqconn

import (
	"context"
	"database/sql"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// New creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/lib/pq as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != "postgres" {
		return nil, fmt.Errorf(`invalid driver %q, pqconn expects "postgres"`, config.Driver)
	}
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
	structFieldNamer sqldb.StructFieldNamer
}

func (conn *connection) WithContext(ctx context.Context) sqldb.Connection {
	return &connection{
		ctx:              ctx,
		db:               conn.db,
		config:           conn.config,
		structFieldNamer: conn.structFieldNamer,
	}
}

func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	return &connection{
		ctx:              conn.ctx,
		db:               conn.db,
		config:           conn.config,
		structFieldNamer: namer,
	}
}

func (conn *connection) StructFieldNamer() sqldb.StructFieldNamer {
	return conn.structFieldNamer
}

func (conn *connection) Stats() sql.DBStats {
	return conn.db.Stats()
}

func (conn *connection) Config() *sqldb.Config {
	return conn.config
}

func (conn *connection) Ping() error {
	return conn.db.PingContext(conn.ctx)
}

func (conn *connection) Exec(query string, args ...interface{}) error {
	_, err := conn.db.ExecContext(conn.ctx, query, args...)
	return impl.WrapNonNilErrorWithQuery(err, query, args)
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
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, args)
		return sqldb.RowScannerWithError(err)
	}
	return impl.NewRowScanner(rows, conn.structFieldNamer, query, args)
}

func (conn *connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, args)
		return sqldb.RowsScannerWithError(err)
	}
	return impl.NewRowsScanner(conn.ctx, rows, conn.structFieldNamer, query, args)
}

func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	tx, err := conn.db.BeginTx(conn.ctx, opts)
	if err != nil {
		return nil, err
	}
	return &transaction{
		connection:       conn,
		tx:               tx,
		structFieldNamer: conn.structFieldNamer,
	}, nil
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
