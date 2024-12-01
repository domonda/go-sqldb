package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ConnectionWithError returns a dummy Connection
// where all methods return the passed error.
func ConnectionWithError(ctx context.Context, err error) Connection {
	if err == nil {
		panic("ConnectionWithError needs an error")
	}
	return connectionWithError{ctx, err}
}

type connectionWithError struct {
	ctx context.Context
	err error
}

func (e connectionWithError) Context() context.Context { return e.ctx }

func (e connectionWithError) WithContext(ctx context.Context) Connection {
	return connectionWithError{ctx: ctx, err: e.err}
}

func (e connectionWithError) WithStructFieldMapper(namer StructFieldMapper) Connection {
	return e
}

func (e connectionWithError) StructFieldMapper() StructFieldMapper {
	return DefaultStructFieldMapping
}

func (e connectionWithError) Ping(time.Duration) error {
	return e.err
}

func (e connectionWithError) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (e connectionWithError) Config() *Config {
	return &Config{Err: e.err}
}

func (e connectionWithError) Placeholder(paramIndex int) string {
	return fmt.Sprintf("?%d", paramIndex+1)
}

func (e connectionWithError) ValidateColumnName(name string) error {
	return e.err
}

func (e connectionWithError) Exec(query string, args ...any) error {
	return e.err
}

func (e connectionWithError) QueryRow(query string, args ...any) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) QueryRows(query string, args ...any) RowsScanner {
	return RowsScannerWithError(e.err)
}

func (e connectionWithError) IsTransaction() bool {
	return false
}

func (e connectionWithError) TransactionNo() uint64 {
	return 0
}

func (ce connectionWithError) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (e connectionWithError) Begin(opts *sql.TxOptions, no uint64) (Connection, error) {
	return nil, e.err
}

func (e connectionWithError) Commit() error {
	return e.err
}

func (e connectionWithError) Rollback() error {
	return e.err
}

func (e connectionWithError) Transaction(opts *sql.TxOptions, txFunc func(tx Connection) error) error {
	return e.err
}

func (e connectionWithError) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return e.err
}

func (e connectionWithError) UnlistenChannel(channel string) error {
	return e.err
}

func (e connectionWithError) IsListeningOnChannel(channel string) bool {
	return false
}

func (e connectionWithError) Close() error {
	return e.err
}
