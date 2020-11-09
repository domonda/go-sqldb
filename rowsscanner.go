package sqldb

import (
	"context"
	"fmt"

	"github.com/domonda/go-sqldb/impl"
)

// RowsScanner scans the values from multiple rows.
type RowsScanner interface {
	// ScanSlice scans one value per row into one slice element of dest.
	// dest must be a pointer to a slice with a row value compatible element type.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// It is an error to query more than one column.
	ScanSlice(dest interface{}) error

	// ScanStructSlice scans every row into the struct fields of dest slice elements.
	// dest must be a pointer to a slice of structs or struct pointers.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// Every mapped struct field must have a corresponding column in the query results.
	ScanStructSlice(dest interface{}) error

	// ScanStrings scans the values of all rows as strings.
	// Byte slices will be interpreted as strings,
	// nil (SQL NULL) will be converted to an empty string,
	// all other types are converted with fmt.Sprint.
	// If true is passed for headerRow, then a row
	// with the column names will be prepended.
	ScanStrings(headerRow bool) (rows [][]string, err error)

	// ForEachRow will call the passed callback with a RowScanner for every row.
	// In case of zero rows, no error will be returned.
	ForEachRow(callback func(RowScanner) error) error

	// ForEachRowCall will call the passed callback with scanned values or a struct for every row.
	// If the callback function has a single struct or struct pointer argument,
	// then RowScanner.ScanStruct will be used per row,
	// else RowScanner.Scan will be used for all arguments of the callback.
	// If the function has a context.Context as first argument,
	// then the context of the query call will be passed on.
	// The callback can have no result or a single error result value.
	// If a non nil error is returned from the callback, then this error
	// is returned immediately by this function without scanning further rows.
	// In case of zero rows, no error will be returned.
	ForEachRowCall(callback interface{}) error
}

type rowsScanner struct {
	ctx              context.Context // ctx is checked for every row and passed through to callbacks
	rows             Rows
	structFieldNamer StructFieldNamer
	query            string        // for error wrapping
	args             []interface{} // for error wrapping
}

func newRowsScanner(ctx context.Context, rows Rows, structFieldNamer StructFieldNamer, query string, args []interface{}) RowsScanner {
	return &rowsScanner{ctx, rows, structFieldNamer, query, args}
}

func (s *rowsScanner) ScanSlice(dest interface{}) error {
	err := impl.ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, impl.FormatQuery(s.query, s.args...))
	}
	return nil
}

func (s *rowsScanner) ScanStructSlice(dest interface{}) error {
	err := impl.ScanRowsAsSlice(s.ctx, s.rows, dest, s.structFieldNamer)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, impl.FormatQuery(s.query, s.args...))
	}
	return nil
}

func (s *rowsScanner) ScanStrings(headerRow bool) (rows [][]string, err error) {
	cols, err := s.rows.Columns()
	if err != nil {
		return nil, err
	}
	if headerRow {
		rows = [][]string{cols}
	}
	stringScannablePtrs := make([]interface{}, len(cols))
	err = s.ForEachRow(func(rowScanner RowScanner) error {
		row := make([]string, len(cols))
		for i := range stringScannablePtrs {
			stringScannablePtrs[i] = (*StringScannable)(&row[i])
		}
		err := rowScanner.Scan(stringScannablePtrs...)
		if err != nil {
			return err
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *rowsScanner) ForEachRow(callback func(RowScanner) error) (err error) {
	defer func() {
		s.rows.Close()
		err = impl.WrapNonNilErrorWithQuery(err, s.query, s.args)
	}()

	for s.rows.Next() {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}

		err := callback(CurrentRowScanner{s.rows, s.structFieldNamer})
		if err != nil {
			return err
		}
	}
	return s.rows.Err()
}

func (s *rowsScanner) ForEachRowCall(callback interface{}) error {
	forEachRowFunc, err := impl.ForEachRowCallFunc(s.ctx, callback)
	if err != nil {
		return err
	}
	return s.ForEachRow(forEachRowFunc)
}
