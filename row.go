package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// Row wraps [Rows] to scan a single row from a query.
type Row struct {
	rows      Rows
	reflector StructReflector
	queryFmt  QueryFormatter // for error wrapping
	query     string         // for error wrapping
	args      []any          // for error wrapping
}

func NewRow(rows Rows, reflector StructReflector, queryFmt QueryFormatter, query string, args []any) *Row {
	return &Row{rows, reflector, queryFmt, query, args}
}

func (r *Row) Columns() ([]string, error) {
	cols, err := r.rows.Columns()
	if err != nil {
		return nil, WrapErrorWithQuery(err, r.query, r.args, r.queryFmt)
	}
	return cols, nil
}

// Scan scans the column values of a row into the passed destination arguments
// following the same logic as [sql.Rows.Scan].
//
// Except when a single destination argument is passed that is a pointer to a struct
// that does not implement sql.Scanner, then the column values of the row
// are scanned into the corresponding struct fields.
func (r *Row) Scan(dest ...any) (err error) {
	defer func() {
		err = errors.Join(err, r.rows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, r.query, r.args, r.queryFmt)
		}
	}()

	if len(dest) == 0 {
		return errors.New("Row.Scan called with no destination arguments")
	}
	isStruct := false
	if len(dest) == 1 {
		v := reflect.ValueOf(dest[0])
		if v.Kind() != reflect.Pointer {
			return fmt.Errorf("Row.Scan destination %T is not a pointer", dest[0])
		}
		if v.IsNil() {
			return fmt.Errorf("Row.Scan destination %T is nil", dest[0])
		}
		v = v.Elem()
		t := v.Type()
		if t.Kind() == reflect.Struct || t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct {
			// dest[0] points to a struct or pointer to struct
			if !t.Implements(typeOfSQLScanner) && !reflect.PointerTo(t).Implements(typeOfSQLScanner) {
				// dest[0] does not implement sql.Scanner
				isStruct = true
			}
		}
	}

	// Check if there was an error even before preparing the row with Next()
	if r.rows.Err() != nil {
		return r.rows.Err()
	}
	if !r.rows.Next() {
		// Error during preparing the row with Next()
		if r.rows.Err() != nil {
			return r.rows.Err()
		}
		return sql.ErrNoRows
	}

	if isStruct {
		cols, err := r.rows.Columns()
		if err != nil {
			return err
		}
		return scanStruct(r.rows, cols, r.reflector, dest[0])
	}

	return r.rows.Scan(dest...)
}

// ScanValues returns the values of a row exactly how they are
// passed from the database driver to an `sql.Scanner`.
// Byte slices will be copied.
func (r *Row) ScanValues() (vals []any, err error) {
	cols, err := r.Columns()
	if err != nil {
		return nil, err // Error is already wrapped with query
	}
	var (
		anys   = make([]AnyValue, len(cols))
		result = make([]any, len(cols))
	)
	// result elements hold pointer to AnyValue for scanning
	for i := range result {
		result[i] = &anys[i]
	}
	err = r.Scan(result...)
	if err != nil {
		return nil, err // Error is already wrapped with query
	}
	// don't return pointers to AnyValue
	// but what internal value has been scanned
	for i := range result {
		result[i] = anys[i].Val
	}
	return result, nil
}

// ScanStrings scans the values of a row as strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
func (r *Row) ScanStrings() (vals []string, err error) {
	cols, err := r.Columns()
	if err != nil {
		return nil, err // Error is already wrapped with query
	}
	var (
		result     = make([]string, len(cols))
		resultPtrs = make([]any, len(cols))
	)
	for i := range resultPtrs {
		resultPtrs[i] = (*StringScannable)(&result[i])
	}
	err = r.Scan(resultPtrs...)
	if err != nil {
		return nil, err // Error is already wrapped with query
	}
	return result, nil
}
