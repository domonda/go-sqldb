package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"
)

// ErrorConnection returns a FullyFeaturedConnection
// where all methods that return errors
// return the passed error.
func ErrorConnection(err error) FullyFeaturedConnection {
	if err == nil {
		panic("nil error for ErrorConnection")
	}
	return errorConnection{err}
}

type errorConnection struct {
	err error
}

func (e errorConnection) Err() error {
	return e.err
}

func (e errorConnection) String() string {
	return e.err.Error()
}

func (e errorConnection) DatabaseKind() string {
	return e.err.Error()
}

func (e errorConnection) StringLiteral(s string) string {
	return defaultQueryFormatter{}.StringLiteral(s)
}

func (e errorConnection) ArrayLiteral(array any) (string, error) {
	return "", e.err
}

func (e errorConnection) ParameterPlaceholder(index int) string {
	return defaultQueryFormatter{}.ParameterPlaceholder(index)
}

func (e errorConnection) ValidateColumnName(name string) error {
	return e.err
}

func (e errorConnection) MapStructField(field reflect.StructField) (table, column string, flags FieldFlag, use bool) {
	return "", "", 0, false
}

func (e errorConnection) MaxParameters() int { return 0 }

func (e errorConnection) DBStats() sql.DBStats {
	return sql.DBStats{}
}

func (e errorConnection) Config() *Config {
	return &Config{Driver: "ErrorConnection"}
}

func (e errorConnection) IsTransaction() bool {
	return false
}

func (e errorConnection) Ping(ctx context.Context, timeout time.Duration) error {
	return errors.Join(e.err, ctx.Err())
}

func (e errorConnection) Exec(ctx context.Context, query string, args ...any) error {
	return errors.Join(e.err, ctx.Err())
}

func (e errorConnection) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return nil, errors.Join(e.err, ctx.Err())
}

func (e errorConnection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

func (e errorConnection) TxNumber() uint64 {
	return 0
}

func (ce errorConnection) TxOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (e errorConnection) Begin(ctx context.Context, opts *sql.TxOptions, no uint64) (TxConnection, error) {
	return nil, errors.Join(e.err, ctx.Err())
}

func (e errorConnection) Commit() error {
	return e.err
}

func (e errorConnection) Rollback() error {
	return e.err
}

func (e errorConnection) ListenOnChannel(ctx context.Context, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return errors.Join(e.err, ctx.Err())
}

func (e errorConnection) UnlistenChannel(ctx context.Context, channel string) error {
	return errors.Join(e.err, ctx.Err())
}

func (e errorConnection) IsListeningOnChannel(ctx context.Context, channel string) bool {
	return false
}

func (e errorConnection) NotifyChannel(ctx context.Context, channel, payload string) error {
	return errors.Join(e.err, ctx.Err())
}

func (e errorConnection) Close() error {
	return e.err
}
