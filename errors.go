package sqldb

import (
	"database/sql"
	"errors"
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
