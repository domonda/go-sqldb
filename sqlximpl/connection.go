package sqlximpl

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/implhelper"
	"github.com/domonda/go-wraperr"
)

func Connection(db *sqlx.DB, driverName, dataSourceName string) sqldb.Connection {
	return &connection{db, driverName, dataSourceName}
}

func NewConnection(ctx context.Context, driverName, dataSourceName string) (sqldb.Connection, error) {
	db, err := sqlx.ConnectContext(ctx, driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return Connection(db, driverName, dataSourceName), nil
}

func MustNewConnection(ctx context.Context, driverName, dataSourceName string) sqldb.Connection {
	conn, err := NewConnection(ctx, driverName, dataSourceName)
	if err != nil {
		panic(err)
	}
	return conn
}

func NewPostgresConnection(ctx context.Context, user, dbname string) (sqldb.Connection, error) {
	driverName := "postgres"
	dataSourceName := fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, dbname)
	return NewConnection(ctx, driverName, dataSourceName)
}

func MustNewPostgresConnection(ctx context.Context, user, dbname string) sqldb.Connection {
	conn, err := NewPostgresConnection(ctx, user, dbname)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	db             *sqlx.DB
	driverName     string
	dataSourceName string
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

// InsertReturning inserts a new row into table using columnValues
// and returns values from the inserted row listed in returning.
func (conn *connection) InsertReturning(table string, columnValues sqldb.Values, returning string) sqldb.RowScanner {
	return implhelper.InsertReturning(context.Background(), conn, table, columnValues, returning)
}

// InsertReturningContext inserts a new row into table using columnValues
// and returns values from the inserted row listed in returning.
func (conn *connection) InsertReturningContext(ctx context.Context, table string, columnValues sqldb.Values, returning string) sqldb.RowScanner {
	return implhelper.InsertReturning(ctx, conn, table, columnValues, returning)
}

func (conn *connection) InsertStruct(table string, rowStruct interface{}, onlyColumns ...string) error {
	return implhelper.InsertStruct(context.Background(), conn, table, rowStruct, onlyColumns...)
}

func (conn *connection) InsertStructContext(ctx context.Context, table string, rowStruct interface{}, onlyColumns ...string) error {
	return implhelper.InsertStruct(ctx, conn, table, rowStruct, onlyColumns...)
}

func (conn *connection) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	return conn.QueryRowContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowContext(ctx context.Context, query string, args ...interface{}) sqldb.RowScanner {
	row := conn.db.QueryRowxContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		err = wraperr.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.NewErrRowScanner(err)
	}
	return &rowScanner{query, row}
}

func (conn *connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.db.QueryxContext(ctx, query, args...)
	if err != nil {
		err = wraperr.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.NewErrRowsScanner(err)
	}
	return &rowsScanner{ctx, query, rows}
}

func (conn *connection) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	tx, err := conn.db.BeginTxx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return TransactionConnection(tx), nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Transaction(txFunc func(tx sqldb.Connection) error) error {
	return implhelper.Transaction(context.Background(), nil, conn, txFunc)
}

func (conn *connection) TransactionContext(ctx context.Context, opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return implhelper.Transaction(ctx, opts, conn, txFunc)
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return getOrCreateGlobalListener(conn.dataSourceName).listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return getGlobalListenerOrNil(conn.dataSourceName).unlistenChannel(channel)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return getGlobalListenerOrNil(conn.dataSourceName).isListeningOnChannel(channel)
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
