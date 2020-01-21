package mockconn

import (
	"context"
	"database/sql"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

type transaction struct {
	*connection
}

func (conn transaction) Begin(ctx context.Context, opts *sql.TxOptions) (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
}

func (conn transaction) Commit() error {
	fmt.Fprintln(conn.queryWriter, "COMMIT")
	return nil
}

func (conn transaction) Rollback() error {
	fmt.Fprintln(conn.queryWriter, "ROLLBACK")
	return nil
}

func (conn transaction) Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return sqldb.ErrWithinTransaction
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