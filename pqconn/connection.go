package pqconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
)

const Driver = "postgres"

// Connect establishes a new sqldb.Connection using the passed sqldb.Config
// and github.com/lib/pq as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
func Connect(ctx context.Context, config *sqldb.ConnConfig) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}

	if config.ReadOnly {
		_, err = db.ExecContext(ctx, `SET default_transaction_read_only = on`)
		if err != nil {
			return nil, fmt.Errorf("failed to set default_transaction_read_only: %w", err)
		}
		// transaction_read_only should be on after setting default_transaction_read_only
		var readOnlyMode string
		err = db.QueryRowContext(ctx, `SHOW transaction_read_only`).Scan(&readOnlyMode)
		if err != nil {
			return nil, fmt.Errorf("failed to check read-only mode: %w", err)
		}
		if readOnlyMode != "on" {
			return nil, errors.New("read-only mode is not enabled")
		}
	}

	return &connection{
		db:     db,
		config: config,
	}, nil
}

// MustConnect creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/lib/pq as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
// Errors are paniced.
func MustConnect(ctx context.Context, config *sqldb.ConnConfig) sqldb.Connection {
	conn, err := Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}

// NewConnExt creates a new sqldb.ConnExt with PostgreSQL-specific components.
// It combines the passed connection and struct reflector with PostgreSQL
// specific QueryFormatter and QueryBuilder.
func NewConnExt(conn sqldb.Connection, structReflector sqldb.StructReflector) *sqldb.ConnExt {
	return sqldb.NewConnExt(
		conn,
		structReflector,
		QueryFormatter{},
		sqldb.StdQueryBuilder{},
	)
}

type connection struct {
	db     *sql.DB
	config *sqldb.ConnConfig
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

func (conn *connection) Exec(ctx context.Context, query string, args ...any) error {
	wrapArrayArgs(args)
	_, err := conn.db.ExecContext(ctx, query, args...)
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (conn *connection) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	wrapArrayArgs(args)
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	return rows
}

func (conn *connection) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	s, err := conn.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return stmt{query, s}, nil
}

func (*connection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelReadCommitted // postgres default
}

func (conn *connection) Transaction() sqldb.TransactionState {
	return sqldb.TransactionState{
		ID:   0,
		Opts: nil,
	}
}

func (conn *connection) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	tx, err := conn.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn, tx, opts, id), nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return conn.getOrCreateListener().listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	return conn.getListenerOrNil().unlistenChannel(channel)
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return conn.getListenerOrNil().isListeningOnChannel(channel)
}

func (conn *connection) Close() error {
	conn.getListenerOrNil().close()
	return conn.db.Close()
}
