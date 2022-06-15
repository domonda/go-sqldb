package sqldb

import (
	"context"
	"fmt"

	"github.com/domonda/go-sqldb/reflection"
)

// RowsScanner scans the values from multiple rows.
type RowsScanner interface {
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

	// Columns returns the column names.
	Columns() ([]string, error)

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
	ForEachRowCall(callback any) error
}

var _ RowsScanner = &rowsScanner{}

// rowsScanner implements rowsScanner with Rows
type rowsScanner struct {
	ctx               context.Context // ctx is checked for every row and passed through to callbacks
	rows              Rows
	structFieldMapper reflection.StructFieldMapper
	query             string // for error wrapping
	argFmt            string // for error wrapping
	args              []any  // for error wrapping
}

func NewRowsScanner(ctx context.Context, rows Rows, structFieldMapper reflection.StructFieldMapper, query, argFmt string, args []any) *rowsScanner {
	return &rowsScanner{ctx, rows, structFieldMapper, query, argFmt, args}
}

func (s *rowsScanner) ScanSlice(dest any) error {
	err := reflection.ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
	}
	return nil
}

func (s *rowsScanner) ScanStructSlice(dest any) error {
	err := reflection.ScanRowsAsSlice(s.ctx, s.rows, dest, s.structFieldMapper)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
	}
	return nil
}

func (s *rowsScanner) Columns() ([]string, error) {
	return s.rows.Columns()
}

func (s *rowsScanner) ScanAllRowsAsStrings(headerRow bool) (rows [][]string, err error) {
	cols, err := s.rows.Columns()
	if err != nil {
		return nil, err
	}
	if headerRow {
		rows = [][]string{cols}
	}
	stringScannablePtrs := make([]any, len(cols))
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
	return rows, err
}

func (s *rowsScanner) ForEachRow(callback func(RowScanner) error) (err error) {
	defer func() {
		err = combineErrors(err, s.rows.Close())
		err = WrapNonNilErrorWithQuery(err, s.query, s.argFmt, s.args)
	}()

	for s.rows.Next() {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}

		err := callback(CurrentRowScanner{s.rows, s.structFieldMapper})
		if err != nil {
			return err
		}
	}
	return s.rows.Err()
}

func (s *rowsScanner) ForEachRowCall(callback any) error {
	forEachRowFunc, err := forEachRowCallFunc(s.ctx, callback)
	if err != nil {
		return err
	}
	return s.ForEachRow(forEachRowFunc)
}
