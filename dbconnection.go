package sqldb

import (
	"context"
	"database/sql"
	"time"
)

type DBConnection struct {
	Conf *Config
	DB   *sql.DB
}

func (c *DBConnection) Config() *Config {
	return c.Conf
}

func (c *DBConnection) Stats() sql.DBStats {
	return c.DB.Stats()
}

func (c *DBConnection) Ping(ctx context.Context, timeout time.Duration) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return c.DB.PingContext(ctx)
}

func (c *DBConnection) Err() error {
	return c.Conf.Err
}

func (c *DBConnection) Exec(ctx context.Context, query string, args ...any) error {
	_, err := c.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, c.Conf.ParamPlaceholderFormatter)
	}
	return nil
}

func (c *DBConnection) QueryRow(ctx context.Context, query string, args ...any) Row {
	rows, err := c.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return RowWithError(WrapErrorWithQuery(err, query, args, c.Conf.ParamPlaceholderFormatter))
	}
	return NewRow(ctx, rows, c, query, args)
}

func (c *DBConnection) QueryRows(ctx context.Context, query string, args ...any) Rows {
	rows, err := c.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return RowsWithError(WrapErrorWithQuery(err, query, args, c.Conf.ParamPlaceholderFormatter))
	}
	return NewRows(ctx, rows, c, query, args)
}

func (c *DBConnection) IsTransaction() bool {
	return false
}

func (c *DBConnection) TxOptions() *sql.TxOptions {
	return nil
}

func (c *DBConnection) Begin(ctx context.Context, opts *sql.TxOptions) (Connection, error) {
	tx, err := c.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &TxConnection{
		Parent: c,
		Tx:     tx,
		Opts:   opts,
	}, nil
}

func (c *DBConnection) Commit() error {
	return ErrNotWithinTransaction
}

func (c *DBConnection) Rollback() error {
	return ErrNotWithinTransaction
}

func (c *DBConnection) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return ErrNotSupported
}

func (c *DBConnection) UnlistenChannel(channel string) error {
	return ErrNotSupported
}

func (c *DBConnection) IsListeningOnChannel(channel string) bool {
	return false
}

func (c *DBConnection) Close() error {
	return c.DB.Close()
}
