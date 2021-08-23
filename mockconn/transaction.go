package mockconn

import (
	"context"
	"database/sql"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

type transaction struct {
	*connection
	opts *sql.TxOptions
}

func (conn transaction) Context() context.Context { return conn.connection.ctx }

func (conn transaction) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == conn.connection.ctx {
		return conn
	}
	return transaction{
		connection: conn.connection.WithContext(ctx).(*connection),
		opts:       conn.opts,
	}
}

func (conn transaction) IsTransaction() bool {
	return true
}

func (conn transaction) TransactionOptions() (*sql.TxOptions, bool) {
	return conn.opts, true
}

func (conn transaction) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "BEGIN")
	}
	return transaction{conn.connection, opts}, nil
}

func (conn transaction) Commit() error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "COMMIT")
	}
	return nil
}

func (conn transaction) Rollback() error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "ROLLBACK")
	}
	return nil
}

func (conn transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn transaction) UnlistenChannel(channel string) (err error) {
	return sqldb.ErrWithinTransaction
}

func (conn transaction) Close() error {
	return conn.Rollback()
}
