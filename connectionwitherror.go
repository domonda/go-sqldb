package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ConnectionWithError returns a dummy FullyFeaturedConnection
// where all methods return the passed error.
func ConnectionWithError(err error) FullyFeaturedConnection {
	if err == nil {
		panic("ConnectionWithError needs an error")
	}
	return connectionWithError{err}
}

type connectionWithError struct {
	err error
}

// Implements DBKind
func (e connectionWithError) DatabaseKind() string {
	return e.err.Error()
}

// Implements DBKind
func (e connectionWithError) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

// Implements DBKind
func (e connectionWithError) ValidateColumnName(name string) error {
	return e.err
}

func (e connectionWithError) DBKind() DBKind {
	return e
}

func (e connectionWithError) DBStats() sql.DBStats {
	return sql.DBStats{}
}

func (e connectionWithError) Config() *Config {
	return &Config{Err: e.err}
}

func (e connectionWithError) IsTransaction() bool {
	return false
}

func (e connectionWithError) Ping(ctx context.Context, timeout time.Duration) error {
	return errors.Join(e.err, ctx.Err())
}

func (e connectionWithError) Now(ctx context.Context) (time.Time, error) {
	return time.Time{}, errors.Join(e.err, ctx.Err())
}

func (e connectionWithError) Exec(ctx context.Context, query string, args ...any) error {
	return errors.Join(e.err, ctx.Err())
}

func (e connectionWithError) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return nil, errors.Join(e.err, ctx.Err())
}

func (e connectionWithError) TxNumber() uint64 {
	return 0
}

func (ce connectionWithError) TxOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (e connectionWithError) Begin(ctx context.Context, opts *sql.TxOptions, no uint64) (TxConnection, error) {
	return nil, errors.Join(e.err, ctx.Err())
}

func (e connectionWithError) Commit() error {
	return e.err
}

func (e connectionWithError) Rollback() error {
	return e.err
}

func (e connectionWithError) ListenOnChannel(ctx context.Context, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return errors.Join(e.err, ctx.Err())
}

func (e connectionWithError) UnlistenChannel(ctx context.Context, channel string) error {
	return errors.Join(e.err, ctx.Err())
}

func (e connectionWithError) IsListeningOnChannel(ctx context.Context, channel string) bool {
	return false
}

func (e connectionWithError) Close() error {
	return e.err
}
