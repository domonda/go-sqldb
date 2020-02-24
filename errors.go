package sqldb

import (
	"database/sql"
	"errors"

	"github.com/domonda/go-wraperr/sentinel"
)

// ErrNoRows

// RemoveErrNoRows returns nil if errors.Is(err, sql.ErrNoRows)
// or else err is returned unchanged.
func RemoveErrNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

// Transaction errors

const (
	ErrWithinTransaction    = sentinel.Error("within a transaction")
	ErrNotWithinTransaction = sentinel.Error("not within a transaction")
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

func (e rowsScannerWithError) ForEachRow(callback func(RowScanner) error) error {
	return e.err
}

func (e rowsScannerWithError) ForEachRowReflect(callback interface{}) error {
	return e.err
}
