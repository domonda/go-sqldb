package sqldb

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNoRows

// RemoveErrNoRows returns nil if errors.Is(err, sql.ErrNoRows)
// or else err is returned unchanged.
func RemoveErrNoRows(err error) error {
	if err == nil || errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

// sentinelError implements the error interface for a string
// and is meant to be used to declare const sentinel errors.
//
// Example:
//   const ErrUserNotFound impl.sentinelError = "user not found"
type sentinelError string

func (s sentinelError) Error() string {
	return string(s)
}

// Transaction errors

const (
	ErrWithinTransaction    sentinelError = "within a transaction"
	ErrNotWithinTransaction sentinelError = "not within a transaction"
)

// ConnectionWithError

// ConnectionWithError returns a dummy Connection
// where all methods return the passed error.
func ConnectionWithError(err error) Connection {
	if err == nil {
		panic("ConnectionWithError(nil) not allowed")
	}
	return connectionWithError{err}
}

type connectionWithError struct {
	err error
}

func (e connectionWithError) WithStructFieldNamer(namer StructFieldNamer) Connection {
	return e
}

func (e connectionWithError) StructFieldNamer() StructFieldNamer {
	return &DefaultStructFieldTagNaming
}

func (e connectionWithError) Ping(ctx context.Context) error {
	return e.err
}

func (e connectionWithError) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (e connectionWithError) Config() *Config {
	return nil
}

func (e connectionWithError) Exec(query string, args ...interface{}) error {
	return e.err
}

func (e connectionWithError) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	return e.err
}

func (e connectionWithError) Insert(table string, values Values) error {
	return e.err
}

func (e connectionWithError) InsertContext(ctx context.Context, table string, values Values) error {
	return e.err
}

func (e connectionWithError) InsertUnique(table string, values Values, onConflict string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertUniqueContext(ctx context.Context, table string, values Values, onConflict string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertReturning(table string, values Values, returning string) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) InsertReturningContext(ctx context.Context, table string, values Values, returning string) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertUniqueStructContext(ctx context.Context, table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertUniqueStructIgnoreColums(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertUniqueStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) Update(table string, values Values, where string, args ...interface{}) error {
	return e.err
}

func (e connectionWithError) UpdateContext(ctx context.Context, table string, values Values, where string, args ...interface{}) error {
	return e.err
}

func (e connectionWithError) UpdateReturningRow(table string, values Values, returning, where string, args ...interface{}) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) UpdateReturningRowContext(ctx context.Context, table string, values Values, returning, where string, args ...interface{}) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) UpdateReturningRows(table string, values Values, returning, where string, args ...interface{}) RowsScanner {
	return RowsScannerWithError(e.err)
}

func (e connectionWithError) UpdateReturningRowsContext(ctx context.Context, table string, values Values, returning, where string, args ...interface{}) RowsScanner {
	return RowsScannerWithError(e.err)
}

func (e connectionWithError) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpdateStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpdateStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpdateStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) QueryRow(query string, args ...interface{}) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) QueryRows(query string, args ...interface{}) RowsScanner {
	return RowsScannerWithError(e.err)
}

func (e connectionWithError) QueryRowsContext(ctx context.Context, query string, args ...interface{}) RowsScanner {
	return RowsScannerWithError(e.err)
}

func (e connectionWithError) IsTransaction() bool {
	return false
}

func (e connectionWithError) Begin(ctx context.Context, opts *sql.TxOptions) (Connection, error) {
	return nil, e.err
}

func (e connectionWithError) Commit() error {
	return e.err
}

func (e connectionWithError) Rollback() error {
	return e.err
}

func (e connectionWithError) Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx Connection) error) error {
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
	if err == nil {
		panic("RowScannerWithError(nil) not allowed")
	}
	return rowScannerWithError{err}
}

type rowScannerWithError struct {
	err error
}

func (e rowScannerWithError) Scan(dest ...interface{}) error {
	return e.err
}

func (e rowScannerWithError) ScanStruct(dest interface{}) error {
	return e.err
}

func (e rowScannerWithError) ScanStrings() ([]string, error) {
	return nil, e.err
}

// RowsScannerWithError

// RowsScannerWithError returns a dummy RowsScanner
// where all methods return the passed error.
func RowsScannerWithError(err error) RowsScanner {
	if err == nil {
		panic("RowsScannerWithError(nil) not allowed")
	}
	return rowsScannerWithError{err}
}

type rowsScannerWithError struct {
	err error
}

func (e rowsScannerWithError) ScanSlice(dest interface{}) error {
	return e.err
}

func (e rowsScannerWithError) ScanStructSlice(dest interface{}) error {
	return e.err
}

func (e rowsScannerWithError) ScanStrings(headerRow bool) ([][]string, error) {
	return nil, e.err
}

func (e rowsScannerWithError) ForEachRow(callback func(RowScanner) error) error {
	return e.err
}

func (e rowsScannerWithError) ForEachRowScan(callback interface{}) error {
	return e.err
}
