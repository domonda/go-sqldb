package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// Rows is an interface with the methods of sql.Rows.
// Allows mocking for tests without an SQL driver.
type Rows interface {
	// Columns returns the column names.
	Columns() ([]string, error)

	// Scan copies the columns in the current row into the values pointed
	// at by dest. The number of values in dest must be the same as the
	// number of columns in Rows.
	Scan(dest ...any) error

	// Close closes the Rows, preventing further enumeration. If Next is called
	// and returns false and there are no further result sets,
	// the Rows are closed automatically and it will suffice to check the
	// result of Err. Close is idempotent and does not affect the result of Err.
	Close() error

	// Next prepares the next result row for reading with the Scan method. It
	// returns true on success, or false if there is no next result row or an error
	// happened while preparing it. Err should be consulted to distinguish between
	// the two cases.
	//
	// Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool

	// Err returns the error, if any, that was encountered during iteration.
	// Err may be called after an explicit or implicit Close.
	Err() error
}

func RowsWithError(err error) Rows {
	return rowsWithError{err}
}

type rowsWithError struct {
	err error
}

func (r rowsWithError) Columns() ([]string, error) { return nil, r.err }
func (r rowsWithError) Scan(dest ...any) error     { return r.err }
func (r rowsWithError) Close() error               { return nil }
func (r rowsWithError) Next() bool                 { return false }
func (r rowsWithError) Err() error                 { return r.err }

// RowsScanner scans the values from multiple rows.
type RowsScanner interface {
	// Columns returns the column names.
	Columns() ([]string, error)

	// ScanSlice scans one value per row into one slice element of dest.
	// dest must be a pointer to a slice with a row value compatible element type.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// It is an error to query more than one column.
	ScanSlice(dest any) error

	// ScanStructSlice scans every row into the struct fields of dest slice elements.
	// dest must be a pointer to a slice of structs or struct pointers.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// Every mapped struct field must have a corresponding column in the query results.
	ScanStructSlice(dest any) error

	// ScanAllRowsAsStrings scans the values of all rows as strings.
	// Byte slices will be interpreted as strings,
	// nil (SQL NULL) will be converted to an empty string,
	// all other types are converted with fmt.Sprint.
	// If true is passed for headerRow, then a row
	// with the column names will be prepended.
	ScanAllRowsAsStrings(headerRow bool) (rows [][]string, err error)
}

// rowsScanner implements RowsScanner with Rows
type rowsScanner struct {
	ctx   context.Context // ctx is checked for every row and passed through to callbacks
	rows  Rows
	query string     // for error wrapping
	args  []any      // for error wrapping
	conn  Connection // for error wrapping
}

func NewRowsScanner(ctx context.Context, rows Rows, query string, args []any, conn Connection) RowsScanner {
	return &rowsScanner{ctx: ctx, rows: rows, query: query, args: args, conn: conn}
}

func (s *rowsScanner) Columns() (columns []string, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return s.rows.Columns()
}

func (s *rowsScanner) ScanSlice(dest any) (err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
}

func (s *rowsScanner) ScanStructSlice(dest any) (err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return ScanRowsAsSlice(s.ctx, s.rows, dest, s.conn)
}

func (s *rowsScanner) ScanAllRowsAsStrings(headerRow bool) (rows [][]string, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	cols, err := s.rows.Columns()
	if err != nil {
		return nil, err
	}
	if headerRow {
		rows = [][]string{cols}
	}
	stringScannablePtrs := make([]any, len(cols))
	for s.rows.Next() {
		if s.ctx.Err() != nil {
			return rows, s.ctx.Err()
		}
		row := make([]string, len(cols))
		for i := range stringScannablePtrs {
			stringScannablePtrs[i] = (*StringScannable)(&row[i])
		}
		err := s.rows.Scan(stringScannablePtrs...)
		if err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}
	return rows, s.rows.Err()
}

// ScanRowsAsSlice scans all srcRows as slice into dest.
// The rows must either have only one column compatible with the element type of the slice,
// or if multiple columns are returned then the slice element type must me a struct or struction pointer
// so that every column maps on exactly one struct field using structFieldMapperOrNil.
// In case of single column rows, nil must be passed for structFieldMapperOrNil.
// ScanRowsAsSlice calls srcRows.Close().
func ScanRowsAsSlice(ctx context.Context, srcRows Rows, dest any, structFieldMapperOrNil StructFieldMapper) error {
	defer srcRows.Close()

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("scan dest is not a pointer but %s", destVal.Type())
	}
	if destVal.IsNil() {
		return errors.New("scan dest is nil")
	}
	slice := destVal.Elem()
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("scan dest is not pointer to slice but %s", destVal.Type())
	}
	sliceElemType := slice.Type().Elem()

	newSlice := reflect.MakeSlice(slice.Type(), 0, 32)

	for srcRows.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		newSlice = reflect.Append(newSlice, reflect.Zero(sliceElemType))
		target := newSlice.Index(newSlice.Len() - 1).Addr()
		if structFieldMapperOrNil != nil {
			err := ScanStruct(srcRows, target.Interface(), structFieldMapperOrNil)
			if err != nil {
				return err
			}
		} else {
			err := srcRows.Scan(target.Interface())
			if err != nil {
				return err
			}
		}
	}
	if srcRows.Err() != nil {
		return srcRows.Err()
	}

	// Assign newSlice if there were no errors
	if newSlice.Len() == 0 {
		slice.SetLen(0)
	} else {
		slice.Set(newSlice)
	}

	return nil
}
