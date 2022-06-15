package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
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

// WrapNonNilErrorWithQuery wraps non nil errors with a formatted query
// if the error was not already wrapped with a query.
// If the passed error is nil, then nil will be returned.
func WrapNonNilErrorWithQuery(err error, query, argFmt string, args []any) error {
	var wrapped errWithQuery
	if err == nil || errors.As(err, &wrapped) {
		return err
	}
	return errWithQuery{err, query, argFmt, args}
}

type errWithQuery struct {
	err    error
	query  string
	argFmt string
	args   []any
}

func (e errWithQuery) Unwrap() error { return e.err }

func (e errWithQuery) Error() string {
	return fmt.Sprintf("%s from query: %s", e.err, FormatQuery(e.query, e.argFmt, e.args...))
}

func combineErrors(prim, sec error) error {
	switch {
	case prim != nil && sec != nil:
		return fmt.Errorf("%w\n%s", prim, sec)
	case prim != nil:
		return prim
	case sec != nil:
		return sec
	}
	return nil
}

// ConnectionWithError
