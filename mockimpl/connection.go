package mockimpl

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/implhelper"
)

func NewConnection(queryWriter io.Writer) sqldb.Connection {
	return &connection{
		queryWriter:      queryWriter,
		listening:        newBoolMap(),
		structFieldNamer: sqldb.DefaultStructFieldTagNaming,
	}
}

type connection struct {
	queryWriter      io.Writer
	listening        *boolMap
	structFieldNamer sqldb.StructFieldNamer
}

// WithStructFieldNamer returns a copy of the connection
// that will use the passed StructFieldNamer.
func (conn *connection) WithStructFieldNamer(namer sqldb.StructFieldNamer) sqldb.Connection {
	return &connection{
		queryWriter:      conn.queryWriter,
		listening:        conn.listening,
		structFieldNamer: namer,
	}
}

func (conn *connection) StructFieldNamer() sqldb.StructFieldNamer {
	return conn.structFieldNamer
}

func (conn *connection) Ping(ctx context.Context) error {
	return nil
}

func (conn *connection) Exec(query string, args ...interface{}) error {
	fmt.Fprintln(conn.queryWriter, query)
	return nil
}

func (conn *connection) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	fmt.Fprintln(conn.queryWriter, query)
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
	fmt.Fprintln(conn.queryWriter, query)
	return new(rowScanner)
}

func (conn *connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	return conn.QueryRowsContext(context.Background(), query, args...)
}

func (conn *connection) QueryRowsContext(ctx context.Context, query string, args ...interface{}) sqldb.RowsScanner {
	fmt.Fprintln(conn.queryWriter, query)
	return new(rowsScanner)
}

func (conn *connection) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	fmt.Fprintln(conn.queryWriter, "BEGIN")
	return transaction{conn}, nil
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
	conn.listening.Set(channel, true)
	fmt.Fprintln(conn.queryWriter, "LISTEN", channel)
	return nil
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	conn.listening.Set(channel, false)
	fmt.Fprintln(conn.queryWriter, "UNLISTEN", channel)
	return nil
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return conn.listening.Get(channel)
}

func (conn *connection) Close() error {
	return nil
}