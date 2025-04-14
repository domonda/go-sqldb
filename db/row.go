package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

// Row wraps [sqldb.Rows] to scan a single row from a query.
type Row struct {
	rows     sqldb.Rows
	queryFmt sqldb.QueryFormatter // for error wrapping
	query    string               // for error wrapping
	args     []any                // for error wrapping
}

func NewRow(rows sqldb.Rows, queryFmt sqldb.QueryFormatter, query string, args []any) *Row {
	return &Row{rows, queryFmt, query, args}
}

func (r *Row) Columns() ([]string, error) {
	cols, err := r.rows.Columns()
	if err != nil {
		return nil, wrapErrorWithQuery(err, r.query, r.args, r.queryFmt)
	}
	return cols, nil
}

func (r *Row) Scan(dest ...any) (err error) {
	defer func() {
		err = errors.Join(err, r.rows.Close())
		if err != nil {
			err = wrapErrorWithQuery(err, r.query, r.args, r.queryFmt)
		}
	}()

	if len(dest) == 0 {
		return errors.New("Row.Scan called with no destination arguments")
	}
	isStruct := false
	if len(dest) == 1 {
		v := reflect.ValueOf(dest[0])
		if v.Kind() != reflect.Ptr {
			return fmt.Errorf("Row.Scan destination %T is not a pointer", dest[0])
		}
		if v.IsNil() {
			return fmt.Errorf("Row.Scan destination %T is nil", dest[0])
		}
		v = v.Elem()
		t := v.Type()
		if t.Kind() == reflect.Struct || t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
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
		return scanStruct(r.rows, cols, defaultStructReflector, dest[0])
	}
	return r.rows.Scan(dest...)
}

// ScanValues returns the values of a row exactly how they are
// passed from the database driver to an `sql.Scanner`.
// Byte slices will be copied.
func (r *Row) ScanValues() (vals []any, err error) {
	cols, err := r.Columns()
	if err != nil {
		return nil, err
	}
	var (
		anys   = make([]sqldb.AnyValue, len(cols))
		result = make([]any, len(cols))
	)
	// result elements hold pointer to sqldb.AnyValue for scanning
	for i := range result {
		result[i] = &anys[i]
	}
	err = r.Scan(result...)
	if err != nil {
		return nil, err
	}
	// don't return pointers to sqldb.AnyValue
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
		return nil, err
	}
	var (
		result     = make([]string, len(cols))
		resultPtrs = make([]any, len(cols))
	)
	for i := range resultPtrs {
		resultPtrs[i] = (*sqldb.StringScannable)(&result[i])
	}
	err = r.Scan(resultPtrs...)
	if err != nil {
		return nil, err
	}
	return result, nil
}
