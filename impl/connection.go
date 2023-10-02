package impl

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// Connection returns a generic sqldb.Connection implementation
// for an existing sql.DB connection.
// argFmt is the format string for argument placeholders like "?" or "$%d"
// that will be replaced error messages to format a complete query.

func (conn *connection) TransactionNo() uint64 {
	return 0
}

func (conn *connection) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (conn *connection) Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error) {
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

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return false
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
