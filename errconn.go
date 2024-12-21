package sqldb

import (
	"context"
	"database/sql"
	"time"
)

// ErrConn implements ListenerConnection
var _ Connection = ErrConn{}

// NewErrConn returns an ErrConn with the passed error.
func NewErrConn(err error) ErrConn {
	if err == nil {
		panic("NewErrConn expects non nil error")
	}
	return ErrConn{Err: err}
}

// ErrConn is a dummy ListenerConnection
// where all methods except Close return Err.
type ErrConn struct {
	StdQueryFormatter

	Err error
}

func (e ErrConn) Ping(context.Context, time.Duration) error {
	return e.Err
}

func (e ErrConn) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (e ErrConn) Config() *Config {
	return &Config{Err: e.Err}
}

func (e ErrConn) Exec(ctx context.Context, query string, args ...any) error {
	return e.Err
}

func (e ErrConn) Query(ctx context.Context, query string, args ...any) Rows {
	return NewErrRows(e.Err)
}

func (ce ErrConn) TransactionInfo() (no uint64, opts *sql.TxOptions) {
	return 0, nil
}

func (e ErrConn) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error) {
	return nil, e.Err
}

func (e ErrConn) Commit() error {
	return e.Err
}

func (e ErrConn) Rollback() error {
	return e.Err
}

func (e ErrConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return e.Err
}

func (e ErrConn) UnlistenChannel(channel string) error {
	return e.Err
}

func (e ErrConn) IsListeningOnChannel(channel string) bool {
	return false
}

func (e ErrConn) Close() error {
	return nil
}
