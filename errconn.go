package sqldb

import (
	"context"
	"database/sql"
	"time"
)

var (
	_ ListenerConnection = ErrConn{}
)

// NewErrConn returns an ErrConn with the passed error.
func NewErrConn(err error) ErrConn {
	if err == nil {
		panic("NewErrConn expects non nil error")
	}
	return ErrConn{Err: err}
}

// ErrConn is a dummy ListenerConnection
// where all methods except Close return Err.
// It embeds StdQueryFormatter to satisfy Connection.
type ErrConn struct {
	StdQueryFormatter

	Err error
}

// Config implements the Connection interface.
func (e ErrConn) Config() *Config {
	return &Config{Driver: "ErrConn"}
}

// Ping implements the Connection interface.
func (e ErrConn) Ping(context.Context, time.Duration) error {
	return e.Err
}

// Stats implements the Connection interface.
func (e ErrConn) Stats() sql.DBStats {
	return sql.DBStats{}
}

// Exec implements the Connection interface.
func (e ErrConn) Exec(ctx context.Context, query string, args ...any) error {
	return e.Err
}

// ExecRowsAffected implements the Connection interface.
func (e ErrConn) ExecRowsAffected(ctx context.Context, query string, args ...any) (int64, error) {
	return 0, e.Err
}

// Query implements the Connection interface.
func (e ErrConn) Query(ctx context.Context, query string, args ...any) Rows {
	return NewErrRows(e.Err)
}

// Prepare implements the Connection interface.
func (e ErrConn) Prepare(ctx context.Context, query string) (Stmt, error) {
	return nil, e.Err
}

// DefaultIsolationLevel implements the Connection interface.
func (e ErrConn) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

// Transaction implements the Connection interface.
func (e ErrConn) Transaction() TransactionState {
	return TransactionState{
		ID:   0,
		Opts: nil,
	}
}

// Begin implements the Connection interface.
func (e ErrConn) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error) {
	return nil, e.Err
}

// Commit implements the Connection interface.
func (e ErrConn) Commit() error {
	return e.Err
}

// Rollback implements the Connection interface.
func (e ErrConn) Rollback() error {
	return e.Err
}

// ListenOnChannel implements the ListenerConnection interface.
func (e ErrConn) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return e.Err
}

// UnlistenChannel implements the ListenerConnection interface.
func (e ErrConn) UnlistenChannel(channel string) error {
	return e.Err
}

// IsListeningOnChannel implements the ListenerConnection interface.
func (e ErrConn) IsListeningOnChannel(channel string) bool {
	return false
}

// Close implements the Connection interface.
func (e ErrConn) Close() error {
	return nil
}
