package sqlxconn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// WithSQLxTx returns a transaction (implementing sqldb.Connection) using a sqldb.Connection and sqlx.Tx.
func WithSQLxTx(conn sqldb.Connection, tx *sqlx.Tx) *transaction {
	return &transaction{
		connection:       conn.(*connection),
		tx:               tx,
		structFieldNamer: conn.StructFieldNamer(),
	}
}

type transaction struct {
	*connection
	tx               *sqlx.Tx
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
	row := conn.tx.QueryRowxContext(ctx, query, args...)
	if row.Err() != nil {
		err := fmt.Errorf("query `%s` returned error: %w", query, row.Err())
		return sqldb.RowScannerWithError(err)
	}
	return &rowScanner{query, row}
}

func (conn *transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn *transaction) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.QueryxContext(ctx, query, args...)
	if err != nil {
		err = fmt.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowsScannerWithError(err)
	}
	return &rowsScanner{ctx, query, rows}
}

func (conn *transaction) IsTransaction() bool {
	return true
}

func (conn *transaction) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
}

func (conn *transaction) Commit() error {
	// fmt.Println("sqlxconn.transaction.Commit")
	return conn.tx.Commit()
}

func (conn *transaction) Rollback() error {
	// fmt.Println("sqlxconn.transaction.Rollback")
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
