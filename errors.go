package sqldb

import (
	"database/sql"
	"errors"
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
	return err != nil && !errors.Is(err, sql.ErrNoRows)
}

// sentinelError implements the error interface for a string
// and is meant to be used to declare const sentinel errors.
//
// Example:
//
//	const ErrUserNotFound impl.sentinelError = "user not found"
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

	ErrNullValueNotAllowed sentinelError = "null value not allowed"
)

type ErrRaisedException struct {
	Message string
}

func (e ErrRaisedException) Error() string {
	return "raised exception: " + e.Message
}

type ErrIntegrityConstraintViolation struct {
	Constraint string
}

func (e ErrIntegrityConstraintViolation) Error() string {
	if e.Constraint == "" {
		return "integrity constraint violation"
	}
	return "integrity constraint violation of constraint: " + e.Constraint
}

type ErrRestrictViolation struct {
	Constraint string
}

func (e ErrRestrictViolation) Error() string {
	if e.Constraint == "" {
		return "restrict violation"
	}
	return "restrict violation of constraint: " + e.Constraint
}

func (e ErrRestrictViolation) Unwrap() error {
	return ErrIntegrityConstraintViolation{Constraint: e.Constraint}
}

type ErrNotNullViolation struct {
	Constraint string
}

func (e ErrNotNullViolation) Error() string {
	if e.Constraint == "" {
		return "not null violation"
	}
	return "not null violation of constraint: " + e.Constraint
}

func (e ErrNotNullViolation) Unwrap() error {
	return ErrIntegrityConstraintViolation{Constraint: e.Constraint}
}

type ErrForeignKeyViolation struct {
	Constraint string
}

func (e ErrForeignKeyViolation) Error() string {
	if e.Constraint == "" {
		return "foreign key violation"
	}
	return "foreign key violation of constraint: " + e.Constraint
}

func (e ErrForeignKeyViolation) Unwrap() error {
	return ErrIntegrityConstraintViolation{Constraint: e.Constraint}
}

type ErrUniqueViolation struct {
	Constraint string
}

func (e ErrUniqueViolation) Error() string {
	if e.Constraint == "" {
		return "unique violation"
	}
	return "unique violation of constraint: " + e.Constraint
}

func (e ErrUniqueViolation) Unwrap() error {
	return ErrIntegrityConstraintViolation{Constraint: e.Constraint}
}

type ErrCheckViolation struct {
	Constraint string
}

func (e ErrCheckViolation) Error() string {
	if e.Constraint == "" {
		return "check violation"
	}
	return "check violation of constraint: " + e.Constraint
}

func (e ErrCheckViolation) Unwrap() error {
	return ErrIntegrityConstraintViolation{Constraint: e.Constraint}
}

type ErrExclusionViolation struct {
	Constraint string
}

func (e ErrExclusionViolation) Error() string {
	if e.Constraint == "" {
		return "exclusion violation"
	}
	return "exclusion violation of constraint: " + e.Constraint
}

func (e ErrExclusionViolation) Unwrap() error {
	return ErrIntegrityConstraintViolation{Constraint: e.Constraint}
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
