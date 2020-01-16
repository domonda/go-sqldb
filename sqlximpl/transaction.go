package sqlximpl

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/implhelper"
)

type transaction struct {
	tx *sqlx.Tx
}

func TransactionConnection(tx *sqlx.Tx) sqldb.Connection {
	return transaction{tx}
}

func (conn transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return err
}

func (conn transaction) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := conn.tx.ExecContext(ctx, query, args...)
	return err
}

// Insert a new row into table using the named columValues.
func (conn transaction) Insert(table string, columValues sqldb.Values) error {
	return implhelper.Insert(context.Background(), conn, table, columValues)
}

func (conn transaction) InsertContext(ctx context.Context, table string, columValues sqldb.Values) error {
	return implhelper.Insert(ctx, conn, table, columValues)
}

// InsertReturning inserts a new row into table using columnValues
// and returns values from the inserted row listed in returning.
func (conn transaction) InsertReturning(table string, columnValues sqldb.Values, returning string) sqldb.RowScanner {
	return implhelper.InsertReturning(context.Background(), conn, table, columnValues, returning)
}

func (conn transaction) InsertReturningContext(ctx context.Context, table string, columnValues sqldb.Values, returning string) sqldb.RowScanner {
	return implhelper.InsertReturning(ctx, conn, table, columnValues, returning)
}

func (conn transaction) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return implhelper.InsertStruct(context.Background(), conn, table, rowStruct, nil, restrictToColumns)
}

func (conn transaction) InsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return implhelper.InsertStruct(ctx, conn, table, rowStruct, nil, restrictToColumns)
}

// InsertStructIgnoreColums inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
func (conn transaction) InsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return implhelper.InsertStruct(context.Background(), conn, table, rowStruct, ignoreColumns, nil)
}

// InsertStructIgnoreColumsContext inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
func (conn transaction) InsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return implhelper.InsertStruct(ctx, conn, table, rowStruct, ignoreColumns, nil)
}

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn transaction) UpsertStruct(table string, rowStruct interface{}, idColumn string, restrictToColumns ...string) error {
	return implhelper.UpsertStruct(context.Background(), conn, table, rowStruct, idColumn, nil, restrictToColumns)
}

// UpsertStructContext upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn transaction) UpsertStructContext(ctx context.Context, table string, rowStruct interface{}, idColumn string, restrictToColumns ...string) error {
	return implhelper.UpsertStruct(ctx, conn, table, rowStruct, idColumn, nil, restrictToColumns)
}

// UpsertStructIgnoreColums upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn transaction) UpsertStructIgnoreColums(table string, rowStruct interface{}, idColumn string, ignoreColumns ...string) error {
	return implhelper.UpsertStruct(context.Background(), conn, table, rowStruct, idColumn, ignoreColumns, nil)
}

// UpsertStructIgnoreColumsContext upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn transaction) UpsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, idColumn string, ignoreColumns ...string) error {
	return implhelper.UpsertStruct(ctx, conn, table, rowStruct, idColumn, ignoreColumns, nil)
}

func (conn transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	return conn.QueryRowContext(context.Background(), query, args...)
}

func (conn transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) sqldb.RowScanner {
	row := conn.tx.QueryRowxContext(ctx, query, args...)
	if row.Err() != nil {
		return sqldb.NewErrRowScanner(row.Err())
	}
	return &rowScanner{query, row}
}

func (conn transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn transaction) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.QueryxContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRowsScanner(err)
	}
	return &rowsScanner{ctx, query, rows}
}

func (conn transaction) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
}

func (conn transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn transaction) Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return sqldb.ErrWithinTransaction
}

func (conn transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return getOrCreateGlobalListener(conn.tx.DriverName()).listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn transaction) UnlistenChannel(channel string) (err error) {
	return getGlobalListenerOrNil(conn.tx.DriverName()).unlistenChannel(channel)
}

func (conn transaction) IsListeningOnChannel(channel string) bool {
	return getGlobalListenerOrNil(conn.tx.DriverName()).isListeningOnChannel(channel)
}

func (conn transaction) Close() error {
	conn.Rollback()
	return nil
}
