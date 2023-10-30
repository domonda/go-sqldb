package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"time"
)

type QueryLogger interface {
	LogQuery(query string, args []any)
}

type QueryLoggerFunc func(query string, args []any)

func (f QueryLoggerFunc) LogQuery(query string, args []any) {
	f(query, args)
}

type QueryBuffer struct {
	b strings.Builder
}

func (b *QueryBuffer) LogQuery(query string, args []any) {
	b.b.WriteString(query)
	b.b.WriteString(";\n")
}

func (b *QueryBuffer) String() string {
	return b.b.String()
}

func (b *QueryBuffer) Len() int {
	return b.b.Len()
}

func (b *QueryBuffer) Reset() {
	b.b.Reset()
}

func LogConnection(target Connection, queryLogger QueryLogger) FullyFeaturedConnection {
	if target == nil {
		panic("<nil> target Connection")
	}
	if queryLogger == nil {
		panic("<nil> queryLogger")
	}
	return &logConnection{
		target:      target,
		queryLogger: queryLogger,
	}
}

type logConnection struct {
	target      Connection
	queryLogger QueryLogger
}

func (c *logConnection) Err() error {
	return c.target.Err()
}

func (c *logConnection) String() string {
	return "LogConnection->" + c.target.String()
}

func (c *logConnection) DatabaseKind() string {
	return c.target.DatabaseKind()
}

func (c *logConnection) StringLiteral(s string) string {
	return c.target.StringLiteral(s)
}

func (c *logConnection) ArrayLiteral(array any) (string, error) {
	return c.target.ArrayLiteral(array)
}

func (c *logConnection) ParameterPlaceholder(index int) string {
	return c.target.ParameterPlaceholder(index)
}

func (c *logConnection) ValidateColumnName(name string) error {
	return c.target.ValidateColumnName(name)
}

func (c *logConnection) MapStructField(field reflect.StructField) (table, column string, flags FieldFlag, use bool) {
	return c.target.MapStructField(field)
}

func (c *logConnection) MaxParameters() int {
	return c.target.MaxParameters()
}

func (c *logConnection) DBStats() sql.DBStats {
	return c.target.DBStats()
}

func (c *logConnection) Config() *Config {
	return c.target.Config()
}

func (c *logConnection) IsTransaction() bool {
	return c.target.IsTransaction()
}

func (c *logConnection) Ping(ctx context.Context, timeout time.Duration) error {
	return c.target.Ping(ctx, timeout)
}

func (c *logConnection) Exec(ctx context.Context, query string, args ...any) error {
	c.queryLogger.LogQuery(query, args)
	return c.target.Exec(ctx, query, args...)
}

func (c *logConnection) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	c.queryLogger.LogQuery(query, args)
	return c.target.Query(ctx, query, args...)
}

func (c *logConnection) DefaultIsolationLevel() sql.IsolationLevel {
	target, ok := c.target.(TxConnection)
	if !ok {
		return sql.LevelDefault
	}
	return target.DefaultIsolationLevel()
}

func (c *logConnection) TxNumber() uint64 {
	target, ok := c.target.(TxConnection)
	if !ok {
		return 0
	}
	return target.TxNumber()
}

func (c *logConnection) TxOptions() (*sql.TxOptions, bool) {
	target, ok := c.target.(TxConnection)
	if !ok {
		return nil, false
	}
	return target.TxOptions()
}

func (c *logConnection) Begin(ctx context.Context, opts *sql.TxOptions, no uint64) (TxConnection, error) {
	target, ok := c.target.(TxConnection)
	if !ok {
		return nil, errors.ErrUnsupported
	}
	query := "BEGIN"
	if opts != nil {
		if opts.Isolation != sql.LevelDefault {
			query += " " + strings.ToUpper(opts.Isolation.String())
		}
		if opts.ReadOnly {
			query += " READ ONLY"
		}
	}
	c.queryLogger.LogQuery(query, nil)
	tx, err := target.Begin(ctx, opts, no)
	if err != nil {
		return nil, err
	}
	return LogConnection(tx, c.queryLogger), nil
}

func (c *logConnection) Commit() error {
	target, ok := c.target.(TxConnection)
	if !ok {
		return errors.ErrUnsupported
	}
	c.queryLogger.LogQuery("COMMIT", nil)
	return target.Commit()
}

func (c *logConnection) Rollback() error {
	target, ok := c.target.(TxConnection)
	if !ok {
		return errors.ErrUnsupported
	}
	c.queryLogger.LogQuery("ROLLBACK", nil)
	return target.Commit()
}

func (c *logConnection) ListenChannel(ctx context.Context, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	target, ok := c.target.(NotificationConnection)
	if !ok {
		return errors.ErrUnsupported
	}
	c.queryLogger.LogQuery("LISTEN "+channel, nil)
	return target.ListenChannel(ctx, channel, onNotify, onUnlisten)
}

func (c *logConnection) UnlistenChannel(ctx context.Context, channel string, onNotify OnNotifyFunc) error {
	target, ok := c.target.(NotificationConnection)
	if !ok {
		return errors.ErrUnsupported
	}
	c.queryLogger.LogQuery("UNLISTEN "+channel, nil)
	return target.UnlistenChannel(ctx, channel, onNotify)
}

func (c *logConnection) IsListeningChannel(ctx context.Context, channel string) bool {
	target, ok := c.target.(NotificationConnection)
	if !ok {
		return false
	}
	return target.IsListeningChannel(ctx, channel)
}

func (c *logConnection) ListeningChannels(ctx context.Context) ([]string, error) {
	target, ok := c.target.(NotificationConnection)
	if !ok {
		return nil, nil
	}
	return target.ListeningChannels(ctx)
}

func (c *logConnection) NotifyChannel(ctx context.Context, channel, payload string) error {
	target, ok := c.target.(NotificationConnection)
	if !ok {
		return errors.ErrUnsupported
	}
	query := "NOTIFY " + channel
	if payload != "" {
		query += " " + c.StringLiteral(payload)
	}
	c.queryLogger.LogQuery(query, nil)
	return target.NotifyChannel(ctx, channel, payload)
}

func (c *logConnection) Close() error {
	return c.target.Close()
}
