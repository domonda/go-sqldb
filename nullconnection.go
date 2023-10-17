package sqldb

import (
	"context"
	"database/sql"
	"reflect"
	"time"
)

func NullConnection(queryFormatter QueryFormatter, structFieldMapper StructFieldMapper) FullyFeaturedConnection {
	return &nullConnection{
		queryFormatter:    queryFormatter,
		structFieldMapper: structFieldMapper,
	}
}

type nullConnection struct {
	queryFormatter    QueryFormatter
	structFieldMapper StructFieldMapper

	txNo   uint64
	txOpts *sql.TxOptions
}

func (c *nullConnection) Err() error {
	return nil
}

func (c *nullConnection) String() string {
	return "NullConnection"
}

func (c *nullConnection) DatabaseKind() string {
	return "NullConnection"
}

func (c *nullConnection) StringLiteral(s string) string {
	if c.queryFormatter == nil {
		return defaultQueryFormatter{}.StringLiteral(s)
	}
	return c.queryFormatter.StringLiteral(s)
}

func (c *nullConnection) ArrayLiteral(array any) (string, error) {
	if c.queryFormatter == nil {
		return defaultQueryFormatter{}.ArrayLiteral(array)
	}
	return c.queryFormatter.ArrayLiteral(array)
}

func (c *nullConnection) ParameterPlaceholder(index int) string {
	if c.queryFormatter == nil {
		return defaultQueryFormatter{}.ParameterPlaceholder(index)
	}
	return c.queryFormatter.ParameterPlaceholder(index)
}

func (c *nullConnection) ValidateColumnName(name string) error {
	if c.queryFormatter == nil {
		return nil
	}
	return c.queryFormatter.ValidateColumnName(name)
}

func (c *nullConnection) MapStructField(field reflect.StructField) (table, column string, flags FieldFlag, use bool) {
	if c.structFieldMapper == nil {
		DefaultStructFieldMapping.MapStructField(field)
	}
	return c.structFieldMapper.MapStructField(field)
}

func (c *nullConnection) MaxParameters() int { return 1000 }

func (c *nullConnection) DBStats() sql.DBStats {
	return sql.DBStats{}
}

func (c *nullConnection) Config() *Config {
	return &Config{Driver: "NullConnection"}
}

func (c *nullConnection) IsTransaction() bool {
	return c.txNo > 0
}

func (c *nullConnection) Ping(ctx context.Context, timeout time.Duration) error {
	return ctx.Err()
}

func (c *nullConnection) Exec(ctx context.Context, query string, args ...any) error {
	return ctx.Err()
}

func (c *nullConnection) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nullRows{}, nil
}

func (c *nullConnection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

func (c *nullConnection) TxNumber() uint64 {
	return c.txNo
}

func (c *nullConnection) TxOptions() (*sql.TxOptions, bool) {
	return c.txOpts, c.IsTransaction()
}

func (c *nullConnection) Begin(ctx context.Context, opts *sql.TxOptions, no uint64) (TxConnection, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if c.IsTransaction() {
		return nil, ErrWithinTransaction
	}
	tx := c
	tx.txNo = no
	tx.txOpts = opts
	return tx, nil
}

func (c *nullConnection) Commit() error {
	if !c.IsTransaction() {
		return ErrNotWithinTransaction
	}
	return nil
}

func (c *nullConnection) Rollback() error {
	if !c.IsTransaction() {
		return ErrNotWithinTransaction
	}
	return nil
}

func (c *nullConnection) ListenOnChannel(ctx context.Context, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return ctx.Err()
}

func (c *nullConnection) UnlistenChannel(ctx context.Context, channel string) error {
	return ctx.Err()
}

func (c *nullConnection) IsListeningOnChannel(ctx context.Context, channel string) bool {
	return false
}

func (c *nullConnection) Close() error {
	return nil
}

type nullRows struct{}

func (r nullRows) Columns() ([]string, error) { return nil, nil }
func (r nullRows) Scan(...any) error          { return nil }
func (r nullRows) Close() error               { return nil }
func (r nullRows) Next() bool                 { return false }
func (r nullRows) Err() error                 { return nil }
