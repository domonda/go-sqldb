package impl

import (
	"errors"
	"fmt"
)

// WrapNonNilErrorWithQuery wraps non nil errors with a formatted query
// if the error was not already wrapped with a query.
// If the passed error is nil, then nil will be returned.
func WrapNonNilErrorWithQuery(err error, query, argFmt string, args []any) error {
	if err == nil {
		return nil
	}
	var wrapped errWithQuery
	if errors.As(err, &wrapped) {
		return err // already wrapped
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
