package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb/reflection"
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

func (e connectionWithError) WithStructFieldMapper(reflection.StructFieldMapper) Connection {
	return e
}

func (e connectionWithError) StructFieldMapper() reflection.StructFieldMapper {
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

func (e connectionWithError) ValidateColumnName(name string) error {
	return e.err
}

func (e connectionWithError) ParamPlaceholder(index int) string {
	return fmt.Sprintf(":%d", index+1)
}

func (e connectionWithError) Err() error {
	return e.err
}

func (e connectionWithError) Now() (time.Time, error) {
	return time.Time{}, e.err
}

func (e connectionWithError) Exec(query string, args ...any) error {
	return e.err
}

func (e connectionWithError) Update(table string, values Values, where string, args ...any) error {
	return e.err
}

func (e connectionWithError) UpdateReturningRow(table string, values Values, returning, where string, args ...any) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) UpdateReturningRows(table string, values Values, returning, where string, args ...any) RowsScanner {
	return RowsScannerWithError(e.err)
}

func (e connectionWithError) UpdateStruct(table string, rowStruct any, ignoreColumns ...ColumnFilter) error {
	return e.err
}

func (e connectionWithError) UpsertStruct(table string, rowStruct any, ignoreColumns ...ColumnFilter) error {
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

func (ce connectionWithError) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (e connectionWithError) Begin(opts *sql.TxOptions) (Connection, error) {
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

// RowScannerWithError

// RowScannerWithError returns a dummy RowScanner
// where all methods return the passed error.
func RowScannerWithError(err error) RowScanner {
	return rowScannerWithError{err}
}

type rowScannerWithError struct {
	err error
}

func (e rowScannerWithError) Scan(dest ...any) error {
	return e.err
}

func (e rowScannerWithError) ScanStruct(dest any) error {
	return e.err
}

func (e rowScannerWithError) ScanValues() ([]any, error) {
	return nil, e.err
}

func (e rowScannerWithError) ScanStrings() ([]string, error) {
	return nil, e.err
}

func (e rowScannerWithError) Columns() ([]string, error) {
	return nil, e.err
}

// RowsScannerWithError

// RowsScannerWithError returns a dummy RowsScanner
// where all methods return the passed error.
func RowsScannerWithError(err error) RowsScanner {
	return rowsScannerWithError{err}
}

type rowsScannerWithError struct {
	err error
}

func (e rowsScannerWithError) ScanSlice(dest any) error {
	return e.err
}

func (e rowsScannerWithError) ScanStructSlice(dest any) error {
	return e.err
}

func (e rowsScannerWithError) Columns() ([]string, error) {
	return nil, e.err
}

func (e rowsScannerWithError) ScanAllRowsAsStrings(headerRow bool) ([][]string, error) {
	return nil, e.err
}

func (e rowsScannerWithError) ForEachRow(callback func(RowScanner) error) error {
	return e.err
}

func (e rowsScannerWithError) ForEachRowCall(callback any) error {
	return e.err
}
