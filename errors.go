package sqldb

import (
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

// RowScannerWithError

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

func (e rowsScannerWithError) ForEachRowScan(callback interface{}) error {
	return e.err
}
