package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/domonda/go-sqldb"
)

// ScanRowsAsSlice scans all srcRows as slice into dest.
//
// The sqlRows must either have only one column compatible with the element type of the slice,
// or in case of multiple columns the slice element type must be a struct or struct pointer
// so that every column maps on exactly one struct field using the passed reflector.
//
// In case of single column rows, nil must be passed for reflector.
//
// The function closes the sqlRows.
//
// TODO two different functions for single column and multi column rows?
func ScanRowsAsSlice(ctx context.Context, sqlRows sqldb.Rows, reflector StructReflector, dest any) error {
	defer sqlRows.Close()

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

	for sqlRows.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		newSlice = reflect.Append(newSlice, reflect.Zero(sliceElemType))
		target := newSlice.Index(newSlice.Len() - 1).Addr()
		if reflector != nil {
			err := scanStruct(sqlRows, reflector, target.Interface())
			if err != nil {
				return err
			}
		} else {
			err := sqlRows.Scan(target.Interface())
			if err != nil {
				return err
			}
		}
	}
	if sqlRows.Err() != nil {
		return sqlRows.Err()
	}

	// Assign newSlice if there were no errors
	if newSlice.Len() == 0 {
		slice.SetLen(0)
	} else {
		slice.Set(newSlice)
	}

	return nil
}

/*
// MultiRowScanner
type MultiRowScanner struct {
	ctx       context.Context // ctx is checked for every row and passed through to callbacks
	rows      sqldb.Rows
	reflector StructReflector
	argFmt    sqldb.PlaceholderFormatter // for error wrapping
	query     string                     // for error wrapping
	args      []any                      // for error wrapping
}

func NewMultiRowScanner(ctx context.Context, rows sqldb.Rows, reflector StructReflector, argFmt sqldb.PlaceholderFormatter, query string, args []any) *MultiRowScanner {
	return &MultiRowScanner{ctx, rows, reflector, argFmt, query, args}
}

func (s *MultiRowScanner) Columns() ([]string, error) {
	cols, err := s.rows.Columns()
	if err != nil {
		return nil, wrapErrorWithQuery(err, s.query, s.args, s.argFmt)
	}
	return cols, nil
}

func (s *MultiRowScanner) ScanSlice(dest any) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
	if err != nil {
		return wrapErrorWithQuery(err, s.query, s.args, s.argFmt)
	}
	return nil
}

// TODO is ScanStructSlice needed besides ScanSlice?
func (s *MultiRowScanner) ScanStructSlice(dest any) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, s.reflector)
	if err != nil {
		return wrapErrorWithQuery(err, s.query, s.args, s.argFmt)
	}
	return nil
}

// func (s *MultiRowScanner) ScanAllRowsAsStrings(headerRow bool) (rows [][]string, err error) {
// 	cols, err := s.rows.Columns()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if headerRow {
// 		rows = [][]string{cols}
// 	}
// 	stringScannablePtrs := make([]any, len(cols))
// 	err = s.ForEachRow(func(rowScanner sqldb.RowScanner) error {
// 		row := make([]string, len(cols))
// 		for i := range stringScannablePtrs {
// 			stringScannablePtrs[i] = (*sqldb.StringScannable)(&row[i])
// 		}
// 		err := rowScanner.Scan(stringScannablePtrs...)
// 		if err != nil {
// 			return err
// 		}
// 		rows = append(rows, row)
// 		return nil
// 	})
// 	return rows, err
// }

// func (s *MultiRowScanner) ForEachRow(callback func(*RowScanner) error) (err error) {
// 	defer func() {
// 		err = errors.Join(err, s.rows.Close())
// 		err = WrapNonNilErrorWithQuery(err, s.query, s.argFmt, s.args)
// 	}()

// 	for s.rows.Next() {
// 		if s.ctx.Err() != nil {
// 			return s.ctx.Err()
// 		}

// 		err := callback(CurrentRowScanner{s.rows, s.reflector})
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return s.rows.Err()
// }

// func (s *MultiRowScanner) ForEachRowCall(callback any) error {
// 	forEachRowFunc, err := ForEachRowCallFunc(s.ctx, callback)
// 	if err != nil {
// 		return err
// 	}
// 	return s.ForEachRow(forEachRowFunc)
// }
*/
