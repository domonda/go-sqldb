package mockconn

import (
	"errors"

	sqldb "github.com/domonda/go-sqldb"
)

type Rows struct {
	rows   []*Row
	cursor int // -1 before first Next, len(rows) after Close or last Next
	closed bool
	err    error
}

func NewRows(columnNamer sqldb.StructFieldNamer, rowStructs ...interface{}) *Rows {
	r := &Rows{
		rows:   make([]*Row, len(rowStructs)),
		cursor: -1,
	}
	for i, rowStruct := range rowStructs {
		r.rows[i] = NewRow(rowStruct, columnNamer)
	}
	return r
}

// Columns returns the column names.
func (r *Rows) Columns() ([]string, error) {
	if len(r.rows) == 0 {
		return nil, errors.New("no rows")
	}
	return r.rows[0].Columns()
}

// Scan copies the columns in the current row into the values pointed
// at by dest. The number of values in dest must be the same as the
// number of columns in Rows.
func (r *Rows) Scan(dest ...interface{}) error {
	switch {
	case r.err != nil:
		return r.err
	case r.closed:
		return errors.New("rows are already closed")
	case r.cursor == -1:
		return errors.New("sql: Scan called without calling Next") // original error message from package database/sql
	case r.cursor == len(r.rows):
		return errors.New("no more rows to scan")
	}
	r.err = r.rows[r.cursor].Scan(dest...)
	return r.err
}

// Close closes the Rows, preventing further enumeration. If Next is called
// and returns false and there are no further result sets,
// the Rows are closed automatically and it will suffice to check the
// result of Err. Close is idempotent and does not affect the result of Err.
func (r *Rows) Close() error {
	r.closed = true
	return nil
}

// Next prepares the next result row for reading with the Scan method. It
// returns true on success, or false if there is no next result row or an error
// happened while preparing it. Err should be consulted to distinguish between
// the two cases.
//
// Every call to Scan, even the first one, must be preceded by a call to Next.
func (r *Rows) Next() bool {
	if r.closed || r.cursor >= len(r.rows) || r.err != nil {
		return false
	}
	r.cursor++
	return true
}

// Err returns the error, if any, that was encountered during iteration.
// Err may be called after an explicit or implicit Close.
func (r *Rows) Err() error {
	return r.err
}
