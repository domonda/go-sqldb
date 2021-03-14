package pqconn

import (
	"context"
	"database/sql"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

type transaction struct {
	*connection
	tx               *sql.Tx
	opts             *sql.TxOptions
	structFieldNamer sqldb.StructFieldNamer
}

func (conn *transaction) WithContext(ctx context.Context) sqldb.Connection {
	return &transaction{
		connection:       conn.connection.WithContext(ctx).(*connection), // TODO better way than type cast?
		tx:               conn.tx,
		opts:             conn.opts,
		structFieldNamer: conn.structFieldNamer,
	}
}

func (conn *transaction) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	return &transaction{
		connection:       conn.connection,
		tx:               conn.tx,
		opts:             conn.opts,
		structFieldNamer: namer,
	}
}

func (conn *transaction) StructFieldNamer() sqldb.StructFieldNamer {
	return conn.structFieldNamer
}

func (conn *transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return impl.WrapNonNilErrorWithQuery(err, query, args)
}

func (conn *transaction) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(conn, table, columValues)
}

func (conn *transaction) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(conn, table, values, returning)
}

func (conn *transaction) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	rows, err := conn.tx.QueryContext(conn.connection.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, args)
		return sqldb.RowScannerWithError(err)
	}
	return impl.NewRowScanner(rows, conn.structFieldNamer, query, args)
}

func (conn *transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.QueryContext(conn.connection.ctx, query, args...)
	if err != nil {
		err = impl.WrapNonNilErrorWithQuery(err, query, args)
		return sqldb.RowsScannerWithError(err)
	}
	return impl.NewRowsScanner(conn.connection.ctx, rows, conn.structFieldNamer, query, args)
}

func (conn *transaction) IsTransaction() bool {
	return true
}

func (conn *transaction) TransactionOptions() (*sql.TxOptions, bool) {
	return conn.opts, true
}

func (conn *transaction) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
}

func (conn *transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn *transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn *transaction) UnlistenChannel(channel string) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn *transaction) Close() error {
	return conn.Rollback()
}
