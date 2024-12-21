package impl

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/domonda/go-sqldb"
)

type transaction struct {
	// The parent non-transaction connection is needed
	// for its ctx, Ping(), Stats(), and Config()
	parent *connection
	tx     *sql.Tx
	opts   *sql.TxOptions
	no     uint64
}

func newTransaction(parent *connection, tx *sql.Tx, opts *sql.TxOptions, no uint64) *transaction {
	return &transaction{
		parent: parent,
		tx:     tx,
		opts:   opts,
		no:     no,
	}
}

func (conn *transaction) clone() *transaction {
	c := *conn
	return &c
}

func (conn *transaction) Ping(ctx context.Context, timeout time.Duration) error {
	return conn.parent.Ping(ctx, timeout)
}
func (conn *transaction) Stats() sql.DBStats    { return conn.parent.Stats() }
func (conn *transaction) Config() *sqldb.Config { return conn.parent.Config() }
func (conn *transaction) Placeholder(paramIndex int) string {
	return conn.parent.Placeholder(paramIndex)
}

func (conn *transaction) ValidateColumnName(name string) error {
	return conn.parent.validateColumnName(name)
}

func (conn *transaction) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.tx.ExecContext(ctx, query, args...)
	return err
}

func (conn *transaction) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	rows, err := conn.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(err)
	}
	return rows
}

// func (conn *transaction) QueryRow(query string, args ...any) sqldb.RowScanner {
// 	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
// 	if err != nil {
// 		err = WrapNonNilErrorWithQuery(err, query, conn.parent.argFmt, args)
// 		return sqldb.RowScannerWithError(err)
// 	}
// 	return NewRowScanner(rows, conn.structFieldNamer, query, conn.parent.argFmt, args)
// }

// func (conn *transaction) QueryRows(query string, args ...any) sqldb.RowsScanner {
// 	rows, err := conn.tx.QueryContext(conn.parent.ctx, query, args...)
// 	if err != nil {
// 		err = WrapNonNilErrorWithQuery(err, query, conn.parent.argFmt, args)
// 		return sqldb.RowsScannerWithError(err)
// 	}
// 	return NewRowsScanner(conn.parent.ctx, rows, conn.structFieldNamer, query, conn.parent.argFmt, args)
// }

func (conn *transaction) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return conn.no, conn.opts
}

func (conn *transaction) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	tx, err := conn.parent.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn.parent, tx, opts, no), nil
}

func (conn *transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn *transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *transaction) Close() error {
	return conn.Rollback()
}
