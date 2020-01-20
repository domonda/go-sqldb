package pqtimpl

import (
	"context"
	"database/sql"
	"sync"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/implhelper"
	"github.com/domonda/go-wraperr"
)

func NewConnection(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &connection{
		db:               db,
		config:           config,
		structFieldNamer: sqldb.DefaultStructFieldTagNaming,
	}, nil
}

func MustNewConnection(ctx context.Context, config *sqldb.Config) sqldb.Connection {
	conn, err := NewConnection(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	db               *sql.DB
	config           *sqldb.Config
	structFieldNamer sqldb.StructFieldNamer

	listener    *listener
	listenerMtx sync.RWMutex
}

// WithStructFieldNamer returns a copy of the connection
// that will use the passed StructFieldNamer.
func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	conn.listenerMtx.Lock()
	defer conn.listenerMtx.Unlock()

	return &connection{
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

func (conn *connection) Ping(ctx context.Context) error {
	return conn.db.PingContext(ctx)
}

func (conn *connection) Exec(query string, args ...interface{}) error {
	_, err := conn.db.Exec(query, args...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query, err)
	}
	return nil
}

func (conn *connection) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := conn.db.ExecContext(ctx, query, args...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query, err)
	}
	return nil
}

// Insert a new row into table using the named columValues.
func (conn *connection) Insert(table string, columValues sqldb.Values) error {
	return implhelper.Insert(context.Background(), conn, table, columValues)
}

func (conn *connection) InsertContext(ctx context.Context, table string, columValues sqldb.Values) error {
	return implhelper.Insert(ctx, conn, table, columValues)
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func (conn *connection) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return implhelper.InsertReturning(context.Background(), conn, table, values, returning)
}

// InsertReturningContext inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func (conn *connection) InsertReturningContext(ctx context.Context, table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return implhelper.InsertReturning(ctx, conn, table, values, returning)
}

func (conn *connection) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return implhelper.InsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) InsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return implhelper.InsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

// InsertStructIgnoreColums inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
func (conn *connection) InsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return implhelper.InsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

// InsertStructIgnoreColumsContext inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
func (conn *connection) InsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return implhelper.InsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *connection) UpsertStruct(table string, rowStruct interface{}, idColumn string, restrictToColumns ...string) error {
	return implhelper.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, idColumn, nil, restrictToColumns)
}

// UpsertStructContext upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *connection) UpsertStructContext(ctx context.Context, table string, rowStruct interface{}, idColumn string, restrictToColumns ...string) error {
	return implhelper.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, idColumn, nil, restrictToColumns)
}

// UpsertStructIgnoreColums upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *connection) UpsertStructIgnoreColums(table string, rowStruct interface{}, idColumn string, ignoreColumns ...string) error {
	return implhelper.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, idColumn, ignoreColumns, nil)
}

// UpsertStructIgnoreColumsContext upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *connection) UpsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, idColumn string, ignoreColumns ...string) error {
	return implhelper.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, idColumn, ignoreColumns, nil)
}

func (conn *connection) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	return conn.QueryRowContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowContext(ctx context.Context, query string, args ...interface{}) sqldb.RowScanner {
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		err = wraperr.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowScannerWithError(err)
	}
	return &rowScanner{query, rows, conn.structFieldNamer}
}

func (conn *connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		err = wraperr.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowsScannerWithError(err)
	}
	return &rowsScanner{ctx, query, rows, conn.structFieldNamer}
}

func (conn *connection) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	tx, err := conn.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &transaction{
		conn:             conn,
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

// Transaction executes txFunc within a database transaction that is passed in as tx Connection.
// The transaction will be rolled back if txFunc returns an error or panics.
// Recovered panics are re-paniced after the transaction was rolled back.
// Transaction returns errors from txFunc or transaction commit errors happening after txFunc.
// Rollback errors are logged with sqldb.ErrLogger.
func (conn *connection) Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return implhelper.Transaction(ctx, opts, conn, txFunc)
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
