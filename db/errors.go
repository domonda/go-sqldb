package db

import (
	"errors"
	"fmt"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// wrapErrorWithQuery wraps an errors with a formatted query
// if the error was not already wrapped with a query.
// If the passed error is nil, then nil will be returned.
func wrapErrorWithQuery(err error, query string, args []any, argFmt sqldb.PlaceholderFormatter) error {
	if err == nil {
		return nil
	}
	var wrapped errWithQuery
	if errors.As(err, &wrapped) {
		return err // already wrapped
	}
	return errWithQuery{err, query, args, argFmt}
}

type errWithQuery struct {
	err    error
	query  string
	args   []any
	argFmt sqldb.PlaceholderFormatter
}

func (e errWithQuery) Unwrap() error { return e.err }

func (e errWithQuery) Error() string {
	return fmt.Sprintf("%s from query: %s", e.err, impl.FormatQuery2(e.query, e.argFmt, e.args...))
}
