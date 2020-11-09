package sqldb

import (
	"context"
	"database/sql"
	"errors"
)

var Conn = func(ctx context.Context) (DB, error) {
	return nil, errors.New("no sqldb.Conn")
}

// DB is an interface with the methods of sql.DB needed for this package.
type DB interface {
	// Ping verifies a connection to the database is still alive,
	// establishing a connection if necessary.
	Ping(ctx context.Context) error

	// Stats returns database statistics.
	Stats() sql.DBStats

	// Close closes the database and prevents new queries from starting.
	// Close then waits for all queries that have started processing on the server
	// to finish.
	Close() error

	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(ctx context.Context, query string, args []interface{}) (Rows, error)

	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(ctx context.Context, query string, args []interface{}) error

	// Begin starts a transaction.
	//
	// The provided context is used until the transaction is committed or rolled back.
	// If the context is canceled, the sql package will roll back
	// the transaction. Tx.Commit will return an error if the context provided to
	// Begin is canceled.
	//
	// The provided TxOptions is optional and may be nil if defaults should be used.
	// If a non-default isolation level is used that the driver doesn't support,
	// an error will be returned.
	Begin(ctx context.Context, opts *sql.TxOptions) (Tx, error)
}

type db struct {
	db *sql.DB
}

func NewDB(sqlDB *sql.DB) DB { return db{sqlDB} }

func (d db) Ping(ctx context.Context) error { return d.db.PingContext(ctx) }
func (d db) Stats() sql.DBStats             { return d.db.Stats() }
func (d db) Close() error                   { return d.db.Close() }

func (d db) Query(ctx context.Context, query string, args []interface{}) (Rows, error) {
	return d.db.QueryContext(ctx, query, args)
}

func (d db) Exec(ctx context.Context, query string, args []interface{}) error {
	_, err := d.db.ExecContext(ctx, query, args)
	return err
}

func (d db) Begin(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := d.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return NewTx(tx), nil
}

type Rows interface {
	// Next prepares the next result row for reading with the Scan method. It
	// returns true on success, or false if there is no next result row or an error
	// happened while preparing it. Err should be consulted to distinguish between
	// the two cases.
	//
	// Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool

	// Err returns the error, if any, that was encountered during iteration.
	// Err may be called after an explicit or implicit Close.
	Err() error

	// Columns returns the column names.
	// Columns returns an error if the rows are closed.
	Columns() ([]string, error)

	// ColumnTypes returns column information such as column type, length,
	// and nullable. Some information may not be available from some drivers.
	ColumnTypes() ([]*sql.ColumnType, error)

	// Scan copies the columns in the current row into the values pointed
	// at by dest. The number of values in dest must be the same as the
	// number of columns in Rows.
	Scan(dest ...interface{}) error

	// Close closes the Rows, preventing further enumeration. If Next is called
	// and returns false and there are no further result sets,
	// the Rows are closed automatically and it will suffice to check the
	// result of Err. Close is idempotent and does not affect the result of Err.
	Close() error
}

type Tx interface {
	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(ctx context.Context, query string, args []interface{}) (Rows, error)

	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(ctx context.Context, query string, args []interface{}) error

	// Commit commits the transaction.
	Commit() error

	// Rollback aborts the transaction.
	Rollback() error
}

type tx struct {
	tx *sql.Tx
}

func NewTx(sqlTx *sql.Tx) Tx { return tx{sqlTx} }

func (t tx) Query(ctx context.Context, query string, args []interface{}) (Rows, error) {
	return t.tx.QueryContext(ctx, query, args)
}

func (t tx) Exec(ctx context.Context, query string, args []interface{}) error {
	_, err := t.tx.ExecContext(ctx, query, args)
	return err
}

func (t tx) Commit() error {
	return t.tx.Commit()
}

func (t tx) Rollback() error {
	return t.tx.Rollback()
}
