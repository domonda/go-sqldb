package db

import (
	"errors"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// wrapErrorWithQuery wraps an errors with a formatted query
// if the error was not already wrapped with a query.
// If the passed error is nil, then nil will be returned.
func wrapErrorWithQuery(err error, query string, args []any, queryFmt sqldb.QueryFormatter) error {
	if err == nil {
		return nil
	}
	var wrapped errWithQuery
	if errors.As(err, &wrapped) {
		return err // already wrapped
	}
	return errWithQuery{err, query, args, queryFmt}
}

type errWithQuery struct {
	err      error
	query    string
	args     []any
	queryFmt sqldb.QueryFormatter
}

func (e errWithQuery) Unwrap() error { return e.err }

func (e errWithQuery) Error() string {
	return fmt.Sprintf("%s from query: %s", e.err, sqldb.FormatQuery(e.queryFmt, e.query, e.args...))
}
