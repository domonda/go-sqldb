package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	_ Connection  = connectionWithError{}
	_ RowScanner  = rowScannerWithError{}
	_ RowsScanner = rowsScannerWithError{}
)

// ReplaceErrNoRows returns the passed replacement error
// if errors.Is(err, sql.ErrNoRows),
// else the passed err is returned unchanged.
func ReplaceErrNoRows(err, replacement error) error {
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return replacement
	}
	return err
}

// IsOtherThanErrNoRows returns true if the passed error is not nil
// and does not unwrap to, or is sql.ErrNoRows.
func IsOtherThanErrNoRows(err error) bool {
	return err != nil && errors.Is(err, sql.ErrNoRows)
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
	// ErrWithinTransaction is returned by methods
	// that are not allowed within DB transactions
	// when the DB connection is a transaction.
	ErrWithinTransaction sentinelError = "within a transaction"

	// ErrNotWithinTransaction is returned by methods
	// that are are only allowed within DB transactions
	// when the DB connection is not a transaction.
	ErrNotWithinTransaction sentinelError = "not within a transaction"

	// ErrNotSupported is returned when a connection
	// does not support a certain method.
	ErrNotSupported sentinelError = "not supported"
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

func (e connectionWithError) Exec(query string, args ...any) error {
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

func (e connectionWithError) InsertStruct(table string, rowStruct any, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertStructNonDefault(table string, rowStruct any) error {
	return e.err
}

func (e connectionWithError) InsertStructIgnoreColumns(table string, rowStruct any, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) InsertUniqueStruct(table string, rowStruct any, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return false, e.err
}

func (e connectionWithError) InsertUniqueStructIgnoreColumns(table string, rowStruct any, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return false, e.err
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

func (e connectionWithError) UpdateStruct(table string, rowStruct any, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpdateStructIgnoreColumns(table string, rowStruct any, ignoreColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStruct(table string, rowStruct any, restrictToColumns ...string) error {
	return e.err
}

func (e connectionWithError) UpsertStructIgnoreColumns(table string, rowStruct any, ignoreColumns ...string) error {
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
