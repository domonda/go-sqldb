package sqlxconn

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/domonda/go-errs"
	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	dbx := sqlx.NewDb(db, config.Driver)
	dbx.MapperFunc(sqldb.DefaultStructFieldTagNaming.UntaggedNameFunc)
	return &connection{
		db:               dbx,
		config:           config,
		structFieldNamer: sqldb.DefaultStructFieldTagNaming,
	}, nil
}

func MustNew(ctx context.Context, config *sqldb.Config) sqldb.Connection {
	conn, err := New(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	db               *sqlx.DB
	config           *sqldb.Config
	structFieldNamer sqldb.StructFieldNamer
}

func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
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
	return nil
}

func (conn *connection) Ping(ctx context.Context) error {
	return conn.db.PingContext(ctx)
}

func (conn *connection) Exec(query string, args ...interface{}) error {
	_, err := conn.db.Exec(query, args...)
	if err != nil {
		return errs.Errorf("query `%s` returned error: %w", query, err)
	}
	return nil
}

func (conn *connection) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := conn.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errs.Errorf("query `%s` returned error: %w", query, err)
	}
	return nil
}

func (conn *connection) Insert(table string, values sqldb.Values) error {
	return impl.Insert(context.Background(), conn, table, values)
}

func (conn *connection) InsertContext(ctx context.Context, table string, values sqldb.Values) error {
	return impl.Insert(ctx, conn, table, values)
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

func (conn *connection) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(context.Background(), conn, table, rowStruct, onConflict, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) InsertUniqueStructContext(ctx context.Context, table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(ctx, conn, table, rowStruct, onConflict, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *connection) InsertUniqueStructIgnoreColums(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(context.Background(), conn, table, rowStruct, onConflict, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *connection) InsertUniqueStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return impl.InsertUniqueStruct(ctx, conn, table, rowStruct, onConflict, conn.structFieldNamer, ignoreColumns, nil)
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
	row := conn.db.QueryRowxContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		err = errs.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowScannerWithError(err)
	}
	return &rowScanner{query, row}
}

func (conn *connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.db.QueryxContext(ctx, query, args...)
	if err != nil {
		err = errs.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowsScannerWithError(err)
	}
	return &rowsScanner{ctx, query, rows}
}

func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	tx, err := conn.db.BeginTxx(ctx, opts)
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

func (conn *connection) Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return impl.Transaction(ctx, opts, conn, txFunc)
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return getOrCreateGlobalListener(conn.config.ConnectURL()).listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return getGlobalListenerOrNil(conn.config.ConnectURL()).unlistenChannel(channel)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return getGlobalListenerOrNil(conn.config.ConnectURL()).isListeningOnChannel(channel)
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
