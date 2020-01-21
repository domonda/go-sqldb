package sqlxconn

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
	"github.com/domonda/go-wraperr"
)

type transaction struct {
	conn             sqldb.Connection
	tx               *sqlx.Tx
	structFieldNamer sqldb.StructFieldNamer
}

func NewTransaction(conn sqldb.Connection, tx *sqlx.Tx) sqldb.Connection {
	return &transaction{
		conn:             conn,
		tx:               tx,
		structFieldNamer: conn.StructFieldNamer(),
	}
}

// WithStructFieldNamer returns a copy of the connection
// that will use the passed StructFieldNamer.
func (conn *transaction) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	return &transaction{
		conn:             conn.conn,
		tx:               conn.tx,
		structFieldNamer: namer,
	}
}

func (conn *transaction) StructFieldNamer() sqldb.StructFieldNamer {
	return conn.structFieldNamer
}

func (conn *transaction) Stats() sql.DBStats {
	return conn.conn.Stats()
}

func (conn *transaction) Config() *sqldb.Config {
	return conn.conn.Config()
}

func (conn *transaction) Ping(ctx context.Context) error {
	return conn.conn.Ping(ctx)
}

func (conn *transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return err
}

func (conn *transaction) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := conn.tx.ExecContext(ctx, query, args...)
	return err
}

// Insert a new row into table using the named columValues.
func (conn *transaction) Insert(table string, columValues sqldb.Values) error {
	return impl.Insert(context.Background(), conn, table, columValues)
}

func (conn *transaction) InsertContext(ctx context.Context, table string, columValues sqldb.Values) error {
	return impl.Insert(ctx, conn, table, columValues)
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
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

// InsertStructIgnoreColums inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
func (conn *transaction) InsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

// InsertStructIgnoreColumsContext inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
func (conn *transaction) InsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.InsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, ignoreColumns, nil)
}

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *transaction) UpsertStruct(table, idColumn string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, idColumn, nil, restrictToColumns)
}

// UpsertStructContext upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *transaction) UpsertStructContext(ctx context.Context, table, idColumn string, rowStruct interface{}, restrictToColumns ...string) error {
	return impl.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, idColumn, nil, restrictToColumns)
}

// UpsertStructIgnoreColums upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *transaction) UpsertStructIgnoreColums(table, idColumn string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(context.Background(), conn, table, rowStruct, conn.structFieldNamer, idColumn, ignoreColumns, nil)
}

// UpsertStructIgnoreColumsContext upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func (conn *transaction) UpsertStructIgnoreColumsContext(ctx context.Context, table, idColumn string, rowStruct interface{}, ignoreColumns ...string) error {
	return impl.UpsertStruct(ctx, conn, table, rowStruct, conn.structFieldNamer, idColumn, ignoreColumns, nil)
}

func (conn *transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	return conn.QueryRowContext(context.Background(), query, args...)
}

func (conn *transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) sqldb.RowScanner {
	row := conn.tx.QueryRowxContext(ctx, query, args...)
	if row.Err() != nil {
		err := wraperr.Errorf("query `%s` returned error: %w", query, row.Err())
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
		err = wraperr.Errorf("query `%s` returned error: %w", query, err)
		return sqldb.RowsScannerWithError(err)
	}
	return &rowsScanner{ctx, query, rows}
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

// Transaction executes txFunc within a database transaction.
// The transaction will be rolled back if txFunc returns an error or panics.
// Recovered panics are re-paniced after the transaction is rolled back.
// Rollback errors are logged with sqldb.ErrLogger.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If inheritConnTx is true and the connection is already a transaction,
// then this transaction is inherited for txFunc ignoring opts and Begin/Commit are not called on the connection.
// Errors or panics will roll back the inherited transaction though.
func (conn *transaction) Transaction(ctx context.Context, opts *sql.TxOptions, inheritConnTx bool, txFunc func(tx sqldb.Connection) error) error {
	return impl.Transaction(ctx, opts, conn, inheritConnTx, txFunc)
}

func (conn *transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn *transaction) UnlistenChannel(channel string) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn *transaction) IsListeningOnChannel(channel string) bool {
	return conn.conn.IsListeningOnChannel(channel)
}

func (conn *transaction) Close() error {
	return conn.Rollback()
}
