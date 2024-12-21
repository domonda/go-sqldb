package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// NewGenericConn returns a generic Connection implementation
// for an existing sql.DB connection.
// argFmt is the format string for argument placeholders like "?" or "$%d"
// that will be replaced error messages to format a complete query.
func NewGenericConn(db *sql.DB, config *Config, validateColumnName func(string) error, argFmt string) Connection {
	return &genericConn{
		db:                 db,
		config:             config,
		argFmt:             argFmt,
		validateColumnName: validateColumnName,
	}
}

type genericConn struct {
	db                 *sql.DB
	config             *Config
	argFmt             string
	validateColumnName func(string) error
}

func (conn *genericConn) Ping(ctx context.Context, timeout time.Duration) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.db.PingContext(ctx)
}

func (conn *genericConn) Stats() sql.DBStats {
	return conn.db.Stats()
}

func (conn *genericConn) Config() *Config {
	return conn.config
}

func (conn *genericConn) Placeholder(paramIndex int) string {
	return fmt.Sprintf(conn.argFmt, paramIndex+1)
}

func (conn *genericConn) ValidateColumnName(name string) error {
	return conn.validateColumnName(name)
}

func (conn *genericConn) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.db.ExecContext(ctx, query, args...)
	return err
}

func (conn *genericConn) Query(ctx context.Context, query string, args ...any) Rows {
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		return NewErrRows(err)
	}
	return rows
}

func (conn *genericConn) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return 0, nil
}

func (conn *genericConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	tx, err := conn.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newGenericTxConn(conn, tx, opts, no), nil
}

func (conn *genericConn) Commit() error {
	return ErrNotWithinTransaction
}

func (conn *genericConn) Rollback() error {
	return ErrNotWithinTransaction
}

func (conn *genericConn) Close() error {
	return conn.db.Close()
}
