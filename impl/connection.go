package impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
)

// Connection returns a generic sqldb.Connection implementation
// for an existing sql.DB connection.
// argFmt is the format string for argument placeholders like "?" or "$%d"
// that will be replaced error messages to format a complete query.
func Connection(ctx context.Context, db *sql.DB, config *sqldb.Config, validateColumnName func(string) error, argFmt string) sqldb.Connection {
	return &connection{
		ctx:                ctx,
		db:                 db,
		config:             config,
		structFieldNamer:   sqldb.DefaultStructFieldMapping,
		argFmt:             argFmt,
		validateColumnName: validateColumnName,
	}
}

type connection struct {
	ctx                context.Context
	db                 *sql.DB
	config             *sqldb.Config
	structFieldNamer   sqldb.StructFieldMapper
	argFmt             string
	validateColumnName func(string) error
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

func (conn *connection) WithStructFieldMapper(namer sqldb.StructFieldMapper) sqldb.Connection {
	c := conn.clone()
	c.structFieldNamer = namer
	return c
}

func (conn *connection) StructFieldMapper() sqldb.StructFieldMapper {
	return conn.structFieldNamer
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

func (conn *connection) Placeholder(paramIndex int) string {
	return fmt.Sprintf(conn.argFmt, paramIndex+1)
}

func (conn *connection) ValidateColumnName(name string) error {
	return conn.validateColumnName(name)
}

func (conn *connection) Exec(query string, args ...any) error {
	_, err := conn.db.ExecContext(conn.ctx, query, args...)
	return err
}

func (conn *connection) Query(query string, args ...any) (sqldb.Rows, error) {
	return conn.db.QueryContext(conn.ctx, query, args...)
}

func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return NewRowScanner(rows, conn.structFieldNamer, query, conn.argFmt, args)
}

func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	return NewRowsScanner(conn.ctx, rows, conn.structFieldNamer, query, conn.argFmt, args)
}

func (conn *connection) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return 0, nil
}

func (conn *connection) Begin(no uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	tx, err := conn.db.BeginTx(conn.ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn, tx, opts, no), nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
