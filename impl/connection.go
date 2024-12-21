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
func Connection(db *sql.DB, config *sqldb.Config, validateColumnName func(string) error, argFmt string) sqldb.Connection {
	return &connection{
		db:                 db,
		config:             config,
		argFmt:             argFmt,
		validateColumnName: validateColumnName,
	}
}

type connection struct {
	db                 *sql.DB
	config             *sqldb.Config
	argFmt             string
	validateColumnName func(string) error
}

func (conn *connection) clone() *connection {
	c := *conn
	return &c
}

func (conn *connection) Ping(ctx context.Context, timeout time.Duration) error {
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

func (conn *connection) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.db.ExecContext(ctx, query, args...)
	return err
}

func (conn *connection) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(err)
	}
	return rows
}

// func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
// 	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
// 	if err != nil {
// 		err = WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
// 		return sqldb.RowScannerWithError(err)
// 	}
// 	return NewRowScanner(rows, conn.structFieldNamer, query, conn.argFmt, args)
// }

// func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
// 	rows, err := conn.db.QueryContext(conn.ctx, query, args...)
// 	if err != nil {
// 		err = WrapNonNilErrorWithQuery(err, query, conn.argFmt, args)
// 		return sqldb.RowsScannerWithError(err)
// 	}
// 	return NewRowsScanner(conn.ctx, rows, conn.structFieldNamer, query, conn.argFmt, args)
// }

func (conn *connection) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return 0, nil
}

func (conn *connection) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	tx, err := conn.db.BeginTx(ctx, opts)
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
