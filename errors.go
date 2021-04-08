package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ReplaceErrNoRows returns the passed replacement error
// if errors.Is(err, sql.ErrNoRows),
// else err is returned unchanged.
func ReplaceErrNoRows(err, replacement error) error {
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return replacement
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

func (e connectionWithError) WithStructFieldNamer(namer StructFieldNamer) Connection {
	return e
}

func (e connectionWithError) StructFieldNamer() StructFieldNamer {
	return &DefaultStructFieldTagNaming
}

func (e connectionWithError) Ping(time.Duration) error {
	return e.err
}

func (e connectionWithError) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (e connectionWithError) Config() *Config {
	return &Config{Driver: "ConnectionWithError"}
}

func (e connectionWithError) Exec(query string, args ...interface{}) error {
	return e.err
}

func (e connectionWithError) Insert(table string, values Values) error {
	return e.err
}

func (e connectionWithError) InsertUnique(table string, values Values, onConflict string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertReturning(table string, values Values, returning string) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertUniqueStructIgnoreColumns(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) Update(table string, values Values, where string, args ...interface{}) error {
	return e.err
}

func (e connectionWithError) UpdateReturningRow(table string, values Values, returning, where string, args ...interface{}) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) UpdateReturningRows(table string, values Values, returning, where string, args ...interface{}) RowsScanner {
	return RowsScannerWithError(e.err)
}

func (e connectionWithError) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) QueryRow(query string, args ...interface{}) RowScanner {
	return RowScannerWithError(e.err)
}

func (e connectionWithError) QueryRows(query string, args ...interface{}) RowsScanner {
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

func (e rowsScannerWithError) ForEachRowCall(callback interface{}) error {
	return e.err
}
