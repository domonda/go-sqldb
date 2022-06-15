package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
)

// New creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != "mysql" {
		return nil, fmt.Errorf(`invalid driver %q, mysqlconn expects "mysql"`, config.Driver)
	}
	config.DefaultIsolationLevel = sql.LevelRepeatableRead // mysql default

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	conn := &connection{
		ctx:               ctx,
		db:                db,
		config:            config,
		structFieldMapper: sqldb.DefaultStructFieldMapping,
		argFmt:            argFmt,
	}
	return conn, nil
}

// MustNew creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
// Errors are paniced.
func MustNew(ctx context.Context, config *sqldb.Config) sqldb.Connection {
	conn, err := New(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	ctx               context.Context
	db                *sql.DB
	config            *sqldb.Config
	structFieldMapper reflection.StructFieldMapper
	argFmt            string
}

func (conn *connection) clone() *connection {
	c := *conn
	return &c
}

func (conn *connection) Context() context.Context { return conn.ctx }

func (conn *connection) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == conn.ctx {
		return conn
	}
	c := conn.clone()
	c.ctx = ctx
	return c
}

func (conn *connection) WithStructFieldMapper(mapper reflection.StructFieldMapper) sqldb.Connection {
	c := conn.clone()
	c.structFieldMapper = mapper
	return c
}

func (conn *connection) StructFieldMapper() reflection.StructFieldMapper {
	return conn.structFieldMapper
}

func (conn *connection) Ping(timeout time.Duration) error {
	ctx := conn.ctx
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.db.PingContext(ctx)
}

func (conn *connection) Stats() sql.DBStats {
	return conn.db.Stats()
}

func (conn *connection) Config() *sqldb.Config {
	return conn.config
}

func (conn *connection) ValidateColumnName(name string) error {
	return validateColumnName(name)
}

func (conn *connection) ArgFmt() string {
	return conn.argFmt
}

func (conn *connection) Err() error {
	return nil
}

func (conn *connection) Now() (now time.Time, err error) {
	err = conn.QueryRow(`select now()`).Scan(&now)
	if err != nil {
		return time.Time{}, err
	}
	return now, nil
}

func (conn *connection) Exec(query string, args ...any) error {
	_, err := conn.db.ExecContext(conn.ctx, query, args...)
	return sqldb.WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
}

func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = sqldb.WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return sqldb.NewRowScanner(rows, conn.structFieldMapper, query, conn.argFmt, args)
}

func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = sqldb.WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	return sqldb.NewRowsScanner(conn.ctx, rows, conn.structFieldMapper, query, conn.argFmt, args)
}

func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (conn *connection) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	tx, err := conn.db.BeginTx(conn.ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn, tx, opts), nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return fmt.Errorf("notifications %w", sqldb.ErrNotSupported)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return fmt.Errorf("notifications %w", sqldb.ErrNotSupported)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return false
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
