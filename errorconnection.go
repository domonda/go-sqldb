package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"
)

var _ FullyFeaturedConnection = ErrorConnection{}

// ErrorConnection is a FullyFeaturedConnection
// where all methods the return errors
// return the Err field of the struct.
type ErrorConnection struct {
	Err error
}

func (e ErrorConnection) String() string {
	return e.Err.Error()
}

func (e ErrorConnection) DatabaseKind() string {
	return e.Err.Error()
}

func (e ErrorConnection) StringLiteral(s string) string {
	return defaultQueryFormatter{}.StringLiteral(s)
}

func (e ErrorConnection) ArrayLiteral(array any) (string, error) {
	return "", e.Err
}

func (e ErrorConnection) ColumnPlaceholder(index int) string {
	return defaultQueryFormatter{}.ColumnPlaceholder(index)
}

func (e ErrorConnection) MapStructField(field reflect.StructField) (table, column string, flags FieldFlag, use bool) {
	return "", "", 0, false
}

func (e ErrorConnection) ValidateColumnName(name string) error {
	return e.Err
}

func (e ErrorConnection) DBStats() sql.DBStats {
	return sql.DBStats{}
}

func (e ErrorConnection) Config() *Config {
	return &Config{}
}

func (e ErrorConnection) IsTransaction() bool {
	return false
}

func (e ErrorConnection) Ping(ctx context.Context, timeout time.Duration) error {
	return errors.Join(e.Err, ctx.Err())
}

func (e ErrorConnection) Exec(ctx context.Context, query string, args ...any) error {
	return errors.Join(e.Err, ctx.Err())
}

func (e ErrorConnection) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return nil, errors.Join(e.Err, ctx.Err())
}

func (e ErrorConnection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

func (e ErrorConnection) TxNumber() uint64 {
	return 0
}

func (ce ErrorConnection) TxOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (e ErrorConnection) Begin(ctx context.Context, opts *sql.TxOptions, no uint64) (TxConnection, error) {
	return nil, errors.Join(e.Err, ctx.Err())
}

func (e ErrorConnection) Commit() error {
	return e.Err
}

func (e ErrorConnection) Rollback() error {
	return e.Err
}

func (e ErrorConnection) ListenOnChannel(ctx context.Context, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return errors.Join(e.Err, ctx.Err())
}

func (e ErrorConnection) UnlistenChannel(ctx context.Context, channel string) error {
	return errors.Join(e.Err, ctx.Err())
}

func (e ErrorConnection) IsListeningOnChannel(ctx context.Context, channel string) bool {
	return false
}

func (e ErrorConnection) Close() error {
	return e.Err
}
