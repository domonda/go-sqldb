package sqldb

import (
	"context"
	"database/sql"
	"time"
)

type TxConnection struct {
	Parent Connection
	Tx     *sql.Tx
	Opts   *sql.TxOptions
}

func (c *TxConnection) Config() *Config {
	return c.Parent.Config()
}

func (c *TxConnection) Stats() sql.DBStats {
	return c.Parent.Stats()
}

func (c *TxConnection) Ping(ctx context.Context, timeout time.Duration) error {
	return c.Parent.Ping(ctx, timeout)
}

func (c *TxConnection) Err() error {
	return c.Parent.Err()
}

func (c *TxConnection) Exec(ctx context.Context, query string, args ...any) error {
	_, err := c.Tx.ExecContext(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, c.Config().ParamPlaceholderFormatter)
	}
	return nil
}

func (c *TxConnection) QueryRow(ctx context.Context, query string, args ...any) Row {
	rows, err := c.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		return RowWithError(WrapErrorWithQuery(err, query, args, c.Config().ParamPlaceholderFormatter))
	}
	return NewRow(ctx, rows, c, query, args)
}

func (c *TxConnection) QueryRows(ctx context.Context, query string, args ...any) Rows {
	rows, err := c.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		return RowsWithError(WrapErrorWithQuery(err, query, args, c.Config().ParamPlaceholderFormatter))
	}
	return NewRows(ctx, rows, c, query, args)
}

func (c *TxConnection) IsTransaction() bool {
	return true
}

func (c *TxConnection) TxOptions() *sql.TxOptions {
	return c.Opts
}

func (c *TxConnection) Begin(ctx context.Context, opts *sql.TxOptions) (Connection, error) {
	return nil, ErrWithinTransaction
}

func (c *TxConnection) Commit() error {
	return c.Tx.Commit()
}

func (c *TxConnection) Rollback() error {
	return c.Tx.Rollback()
}

func (c *TxConnection) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return ErrWithinTransaction
}

func (c *TxConnection) UnlistenChannel(channel string) error {
	return ErrWithinTransaction
}

func (c *TxConnection) IsListeningOnChannel(channel string) bool {
	return false
}

func (c *TxConnection) Close() error {
	return c.Tx.Rollback()
}
