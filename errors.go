package sqldb

import (
	"database/sql"

	"github.com/domonda/errors"
)

// ErrNoRows

// IsErrNoRows returns if the root cause error of err is sql.ErrNoRows
func IsErrNoRows(err error) bool {
	return errors.Cause(err) == sql.ErrNoRows
}

// FilterErrNoRows returns err or nil if IsErrNoRows(err)
func FilterErrNoRows(err error) error {
	if IsErrNoRows(err) {
		return nil
	}
	return err
}

// Errors considering transactions

var (
	ErrWithinTransaction    = errors.New("within a transaction")
	ErrNotWithinTransaction = errors.New("not within a transaction")
)

// ErrConnection

type ErrConnection struct {
	err error
}

func NewErrConnection(err error) ErrConnection {
	return ErrConnection{err}
}

// Error implements the error interface
func (e ErrConnection) Error() string {
	return e.err.Error()
}

// Unwrap implements xerrors.Wrapper
func (e ErrConnection) Unwrap() error {
	return e.err
}

// Cause implements the unexported causer interface used by errors.Cause.
// Note: Will be removed after transition to xerrors, see Unwrap.
func (e ErrConnection) Cause() error {
	return e.err
}

func (e ErrConnection) Exec(query string, args ...interface{}) error {
	return e
}

func (e ErrConnection) QueryRow(query string, args ...interface{}) RowScanner {
	return ErrRowScanner{e}
}

func (e ErrConnection) QueryRows(query string, args ...interface{}) RowsScanner {
	return ErrRowsScanner{e}
}

func (e ErrConnection) Begin() (Connection, error) {
	return nil, e
}

func (e ErrConnection) Commit() error {
	return e
}

func (e ErrConnection) Rollback() error {
	return e
}

func (e ErrConnection) Transaction(func(tx Connection) error) error {
	return e
}

// ErrRowScanner

type ErrRowScanner struct {
	err error
}

func NewErrRowScanner(err error) ErrRowScanner {
	return ErrRowScanner{err}
}

// Error implements the error interface
func (e ErrRowScanner) Error() string {
	return e.err.Error()
}

// Unwrap implements xerrors.Wrapper
func (e ErrRowScanner) Unwrap() error {
	return e.err
}

// Cause implements the unexported causer interface used by errors.Cause.
// Note: Will be removed after transition to xerrors, see Unwrap.
func (e ErrRowScanner) Cause() error {
	return e.err
}

func (e ErrRowScanner) Scan(dest ...interface{}) error {
	return e
}

func (e ErrRowScanner) ScanStruct(dest interface{}) error {
	return e
}

// ErrRowsScanner

type ErrRowsScanner struct {
	err error
}

func NewErrRowsScanner(err error) ErrRowsScanner {
	return ErrRowsScanner{err}
}

// Error implements the error interface
func (e ErrRowsScanner) Error() string {
	return e.err.Error()
}

// Unwrap implements xerrors.Wrapper
func (e ErrRowsScanner) Unwrap() error {
	return e.err
}

// Cause implements the unexported causer interface used by errors.Cause.
// Note: Will be removed after transition to xerrors, see Unwrap.
func (e ErrRowsScanner) Cause() error {
	return e.err
}

func (e ErrRowsScanner) ScanSlice(dest interface{}) error {
	return e
}

func (e ErrRowsScanner) ScanStructSlice(dest interface{}) error {
	return e
}

func (e ErrRowsScanner) ForEachRow(callback func(RowScanner) error) error {
	return e
}
