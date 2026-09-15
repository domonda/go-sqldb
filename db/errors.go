package db

import "github.com/domonda/go-sqldb"

// ReplaceErrNoRows returns the passed replacement error
// if errors.Is(err, sql.ErrNoRows),
// else the passed err is returned unchanged.
func ReplaceErrNoRows(err, replacement error) error {
	return sqldb.ReplaceErrNoRows(err, replacement)
}

// IsOtherThanErrNoRows returns true if the passed error is not nil
// and does not unwrap to, or is sql.ErrNoRows.
func IsOtherThanErrNoRows(err error) bool {
	return sqldb.IsOtherThanErrNoRows(err)
}

// UnwrapErrorWithQuery returns the error that was wrapped with a query
// by [sqldb.WrapErrorWithQuery], or the passed err unchanged if it was
// not wrapped with a query anywhere in its error chain.
// Passing nil returns nil.
//
// Any error wrapping around the query wrapping is discarded together
// with it, see [sqldb.UnwrapErrorWithQuery].
func UnwrapErrorWithQuery(err error) error {
	return sqldb.UnwrapErrorWithQuery(err)
}
