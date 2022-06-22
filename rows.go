package sqldb

import (
	"context"
	"database/sql"
)

type Rows interface {
	// ForEachRow will call the passed callback with a RowScanner for every row.
	// In case of zero rows, no error will be returned.
	ForEachRow(callback func(Row) error) error

	// Close closes the Rows, preventing further enumeration. If Next is called
	// and returns false and there are no further result sets,
	// the Rows are closed automatically and it will suffice to check the
	// result of Err. Close is idempotent and does not affect the result of Err.
	Close() error
}

///////////////////////////////////////////////////////////////////////////////

// RowsWithError returns dummy Rows
// where all methods return the passed error.
func RowsWithError(err error) Rows {
	return rowsWithError{err}
}

type rowsWithError struct{ err error }

func (e rowsWithError) ForEachRow(func(Row) error) error { return e.err }
func (e rowsWithError) Close() error                     { return e.err }

///////////////////////////////////////////////////////////////////////////////

type rowsWrapper struct {
	ctx   context.Context // ctx is checked for every row and passed through to callbacks
	rows  *sql.Rows
	conn  Connection // for error wrapping
	query string     // for error wrapping
	args  []any      // for error wrapping
}

func NewRows(ctx context.Context, rows *sql.Rows, conn Connection, query string, args []any) Rows {
	return &rowsWrapper{ctx, rows, conn, query, args}
}

func (r *rowsWrapper) ForEachRow(callback func(Row) error) (err error) {
	defer func() {
		err = combineTwoErrors(err, r.rows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, r.query, r.args, r.conn.Config().ParamPlaceholderFormatter)
		}
	}()

	for r.rows.Next() {
		if r.ctx.Err() != nil {
			return r.ctx.Err()
		}

		err := callback(r.rows)
		if err != nil {
			return err
		}
	}
	return r.rows.Err()
}

func (r *rowsWrapper) Close() error {
	return r.rows.Close()
}

///////////////////////////////////////////////////////////////////////////////

// RowAsRows returns a single Rows wrapped as a Rows implementation.
// func RowAsRows(row Row) Rows {
// 	return &rowAsRows{row: row, closed: false}
// }

// type rowAsRows struct {
// 	row    Row
// 	closed bool
// }

// func (r *rowAsRows) ForEachRow(callback func(Row) error) error {
// 	if r.closed {
// 		return errors.New("Rows are closed")
// 	}
// 	err := callback(r.row)
// 	r.closed = true
// 	return err
// }

// func (r *rowAsRows) Close() error {
// 	r.closed = true
// 	return nil
// }
