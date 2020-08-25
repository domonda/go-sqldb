package pqconn

import (
	"context"
	"database/sql"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

type transaction struct {
	*connection
	tx               *sql.Tx
	structFieldNamer sqldb.StructFieldNamer
}

func (conn *transaction) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	return &transaction{
		connection:       conn.connection,
		tx:               conn.tx,
		structFieldNamer: namer,
	}
}

func (conn *transaction) StructFieldNamer() sqldb.StructFieldNamer {
	return conn.structFieldNamer
}

func (conn *transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return err
}

func (conn *transaction) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := conn.tx.ExecContext(ctx, query, args...)
	return err
}

func (conn *transaction) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(context.Background(), conn, table, columValues)
}

func (conn *transaction) InsertContext(ctx context.Context, table string, columValues sqldb.Values) error {
	return impl.Insert(ctx, conn, table, columValues)
}

func (conn *transaction) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(context.Background(), conn, table, values, returning)
}

func (conn *transaction) InsertReturningContext(ctx context.Context, table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return impl.InsertReturning(ctx, conn, table, values, returning)
}

func (conn *transaction) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) InsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.InsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) InsertStructIgnoreColumnsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) UpdateStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpdateStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) UpdateStructIgnoreColumnsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpdateStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) UpsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, nil, restrictToColumns)
}

func (conn *transaction) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) UpsertStructIgnoreColumnsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

func (conn *transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	return conn.QueryRowContext(context.Background(), query, args...)
}

func (conn *transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) sqldb.RowScanner {
	rows, err := conn.tx.QueryContext(ctx, query, args...)
	if err != nil {
		err = fmt.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowScannerWithError(err)
	}
	return &impl.RowScanner{Query: query, Rows: rows, StructFieldNamer: conn.structFieldNamer}
}

func (conn *transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn *transaction) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.QueryContext(ctx, query, args...)
	if err != nil {
		err = fmt.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowsScannerWithError(err)
	}
	return &impl.RowsScanner{Context: ctx, Query: query, Rows: rows, StructFieldNamer: conn.structFieldNamer}
}

// IsTransaction returns if the connection is a transaction
func (conn *transaction) IsTransaction() bool {
	return true
}

func (conn *transaction) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
}

func (conn *transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn *transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *transaction) Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return impl.Transaction(ctx, opts, conn, txFunc)
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
