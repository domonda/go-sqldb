package impl

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

var _ sqldb.RowsScanner = &RowsScanner{}

// RowsScanner implements sqldb.RowsScanner with Rows
type RowsScanner struct {
	ctx              context.Context // ctx is checked for every row and passed through to callbacks
	rows             Rows
	structFieldNamer sqldb.StructFieldMapper
	query            string // for error wrapping
	argFmt           string // for error wrapping
	args             []any  // for error wrapping
}

func NewRowsScanner(ctx context.Context, rows Rows, structFieldNamer sqldb.StructFieldMapper, query, argFmt string, args []any) *RowsScanner {
	return &RowsScanner{ctx, rows, structFieldNamer, query, argFmt, args}
}

func (s *RowsScanner) ScanSlice(dest any) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
	}
	return nil
}

func (s *RowsScanner) ScanStructSlice(dest any) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, s.structFieldNamer)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
	}
	return nil
}

func (s *RowsScanner) Columns() ([]string, error) {
	return s.rows.Columns()
}

func (s *RowsScanner) ScanAllRowsAsStrings(headerRow bool) (rows [][]string, err error) {
	cols, err := s.rows.Columns()
	if err != nil {
		return nil, err
	}
	if headerRow {
		rows = [][]string{cols}
	}
	stringScannablePtrs := make([]any, len(cols))
	err = s.ForEachRow(func(rowScanner sqldb.RowScanner) error {
		row := make([]string, len(cols))
		for i := range stringScannablePtrs {
			stringScannablePtrs[i] = (*sqldb.StringScannable)(&row[i])
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

func (s *RowsScanner) ForEachRow(callback func(sqldb.RowScanner) error) (err error) {
	defer func() {
		err = combineErrors(err, s.rows.Close())
		err = WrapNonNilErrorWithQuery(err, s.query, s.argFmt, s.args)
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

func (s *RowsScanner) ForEachRowCall(callback any) error {
	forEachRowFunc, err := ForEachRowCallFunc(s.ctx, callback)
	if err != nil {
		return err
	}
	return s.ForEachRow(forEachRowFunc)
}

// ScanRowsAsSlice scans all srcRows as slice into dest.
// The rows must either have only one column compatible with the element type of the slice,
// or if multiple columns are returned then the slice element type must me a struct or struction pointer
// so that every column maps on exactly one struct field using structFieldNamer.
// In case of single column rows, nil must be passed for structFieldNamer.
// ScanRowsAsSlice calls srcRows.Close().
func ScanRowsAsSlice(ctx context.Context, srcRows Rows, dest any, structFieldNamer sqldb.StructFieldMapper) error {
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
		if structFieldNamer != nil {
			err := ScanStruct(srcRows, target.Interface(), structFieldNamer)
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
