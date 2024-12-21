package mockconn

import (
	"errors"
	"reflect"
)

type Rows struct {
	rows   []*Row
	cursor int // -1 before first Next, len(rows) after Close or last Next
	closed bool
	err    error
}

func NewRowsFromStructs(rowStructs any, columnNamer StructReflector) *Rows {
	v := reflect.ValueOf(rowStructs)
	t := v.Type()
	if t.Kind() != reflect.Array && t.Kind() != reflect.Slice {
		panic("rowStructs must be array or slice of structs, but is " + t.String())
	}
	if t.Elem().Kind() != reflect.Struct && !(t.Elem().Kind() == reflect.Ptr && t.Elem().Elem().Kind() == reflect.Struct) {
		panic("rowStructs element type must be struct or struct pointer, but is " + t.Elem().String())
	}

	r := &Rows{cursor: -1}
	for i := 0; i < v.Len(); i++ {
		r.rows = append(r.rows, NewRow(v.Index(i).Interface(), columnNamer))
	}
	return r
}

func NewRows(rows ...*Row) *Rows {
	return &Rows{rows: rows, cursor: -1}
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
func (r *Rows) Scan(dest ...any) error {
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
	if r.closed || r.cursor >= len(r.rows)-1 || r.err != nil {
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

// Reset to the state after NewRows (no error, not closed, before first row).
func (r *Rows) Reset() {
	r.cursor = -1
	r.closed = false
	r.err = nil
}
