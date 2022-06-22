package sqldb

import (
	"context"
	"database/sql"
	"time"
)

// ConnectionWithError returns a dummy Connection
// where all methods return the passed error.
func ConnectionWithError(err error) Connection {
	if err == nil {
		panic("ConnectionWithError needs an error")
	}
	return errConn{err}
}

type errConn struct {
	err error
}

func (e errConn) Config() *Config {
	return &Config{Err: e.err}
}

func (e errConn) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (e errConn) Ping(context.Context, time.Duration) error {
	return e.err
}

func (e errConn) Err() error {
	return e.err
}

func (e errConn) Exec(ctx context.Context, query string, args ...any) error {
	return e.err
}

func (e errConn) QueryRow(ctx context.Context, query string, args ...any) Row {
	return RowWithError(e.err)
}

func (e errConn) QueryRows(ctx context.Context, query string, args ...any) Rows {
	return RowsWithError(e.err)
}

func (e errConn) IsTransaction() bool {
	return false
}

func (ce errConn) TxOptions() *sql.TxOptions {
	return nil
}

func (e errConn) Begin(ctx context.Context, opts *sql.TxOptions) (Connection, error) {
	return nil, e.err
}

func (e errConn) Commit() error {
	return e.err
}

func (e errConn) Rollback() error {
	return e.err
}

func (e errConn) Transaction(opts *sql.TxOptions, txFunc func(tx Connection) error) error {
	return e.err
}

func (e errConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return e.err
}

func (e errConn) UnlistenChannel(channel string) error {
	return e.err
}

func (e errConn) IsListeningOnChannel(channel string) bool {
	return false
}

func (e errConn) Close() error {
	return e.err
}
