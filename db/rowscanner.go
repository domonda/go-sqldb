package db

import (
	"database/sql"
	"errors"

	sqldb "github.com/domonda/go-sqldb"
)

// RowScanner implements sqldb.RowScanner for a sql.Row
type RowScanner struct {
	rows      sqldb.Rows
	reflector StructReflector      // for ScanStruct
	queryFmt  sqldb.QueryFormatter // for error wrapping
	query     string               // for error wrapping
	args      []any                // for error wrapping
}

func NewRowScanner(rows sqldb.Rows, reflector StructReflector, queryFmt sqldb.QueryFormatter, query string, args []any) *RowScanner {
	return &RowScanner{rows, reflector, queryFmt, query, args}
}

func (s *RowScanner) Columns() ([]string, error) {
	cols, err := s.rows.Columns()
	if err != nil {
		return nil, wrapErrorWithQuery(err, s.query, s.args, s.queryFmt)
	}
	return cols, nil
}

func (s *RowScanner) Scan(dest ...any) (err error) {
	defer func() {
		err = errors.Join(err, s.rows.Close())
		if err != nil {
			err = wrapErrorWithQuery(err, s.query, s.args, s.queryFmt)
		}
	}()

	if len(dest) == 0 {
		return errors.New("RowScanner.Scan called with no destination arguments")
	}
	// Check if there was an error even before preparing the row with Next()
	if s.rows.Err() != nil {
		return s.rows.Err()
	}
	if !s.rows.Next() {
		// Error during preparing the row with Next()
		if s.rows.Err() != nil {
			return s.rows.Err()
		}
		return sql.ErrNoRows
	}

	return s.rows.Scan(dest...)
}

// // TODO integrate ScanStruct into Scan ?
// func (s *RowScanner) ScanStruct(dest any) (err error) {
// 	defer func() {
// 		err = errors.Join(err, s.rows.Close())
// 		if err != nil {
// 			err = wrapErrorWithQuery(err, s.query, s.args, s.queryFmt)
// 		}
// 	}()

// 	// Check if there was an error even before preparing the row with Next()
// 	if s.rows.Err() != nil {
// 		return s.rows.Err()
// 	}
// 	if !s.rows.Next() {
// 		// Error during preparing the row with Next()
// 		if s.rows.Err() != nil {
// 			return s.rows.Err()
// 		}
// 		return sql.ErrNoRows
// 	}

// 	return scanStruct(s.rows, s.reflector, dest)
// }

// ScanValues returns the values of a row exactly how they are
// passed from the database driver to an `sql.Scanner`.
// Byte slices will be copied.
func (s *RowScanner) ScanValues() (vals []any, err error) {
	cols, err := s.Columns()
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
	err = s.Scan(result...)
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
func (s *RowScanner) ScanStrings() (vals []string, err error) {
	cols, err := s.Columns()
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
	err = s.Scan(resultPtrs...)
	if err != nil {
		return nil, err
	}
	return result, nil
}
