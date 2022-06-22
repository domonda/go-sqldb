package reflection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/domonda/go-types/nullable"
)

// // TODO doc
// // ScanSlice scans one value per row into one slice element of dest.
// // dest must be a pointer to a slice with a row value compatible element type.
// // In case of zero rows, dest will be set to nil and no error will be returned.
// // In case of an error, dest will not be modified.
// // It is an error to query more than one column.func (s *rowsScanner) ScanSlice(dest any) error {
// 	err := reflection.ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
// 	if err != nil {
// 		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
// 	}
// 	return nil
// }
// 	// ScanStructSlice scans every row into the struct fields of dest slice elements.
// 	// dest must be a pointer to a slice of structs or struct pointers.
// 	// In case of zero rows, dest will be set to nil and no error will be returned.
// 	// In case of an error, dest will not be modified.
// 	// Every mapped struct field must have a corresponding column in the query results.
// func (s *rowsScanner) ScanStructSlice(dest any) error {
// 	err := reflection.ScanRowsAsSlice(s.ctx, s.rows, dest, s.structFieldMapper)
// 	if err != nil {
// 		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
// 	}
// 	return nil
// }

// // ScanAllRowsAsStrings scans the values of all rows as strings.
// // Byte slices will be interpreted as strings,
// // nil (SQL NULL) will be converted to an empty string,
// // all other types are converted with fmt.Sprint.
// // If true is passed for headerRow, then a row
// // with the column names will be prepended.
// func (s *rowsScanner) ScanAllRowsAsStrings(headerRow bool) (rows [][]string, err error) {
// 	cols, err := s.rows.Columns()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if headerRow {
// 		rows = [][]string{cols}
// 	}
// 	stringScannablePtrs := make([]any, len(cols))
// 	err = s.ForEachRow(func(rowScanner RowScanner) error {
// 		row := make([]string, len(cols))
// 		for i := range stringScannablePtrs {
// 			stringScannablePtrs[i] = (*StringScannable)(&row[i])
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

// ScanRowsAsSlice scans all srcRows as slice into dest.
// The rows must either have only one column compatible with the element type of the slice,
// or if multiple columns are returned then the slice element type must me a struct or struction pointer
// so that every column maps on exactly one struct field using structFieldMapper.
// In case of single column rows, nil must be passed for structFieldMapper.
// ScanRowsAsSlice calls srcRows.Close().
func ScanRowsAsSlice(ctx context.Context, srcRows Rows, dest any, structFieldMapper StructFieldMapper) error {
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
		if structFieldMapper != nil {
			err := ScanStruct(srcRows, target.Interface(), structFieldMapper)
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

type SliceScanner struct {
	destSlice reflect.Value
}

func WrapWithSliceScanner(destPtr any) any {
	v := reflect.ValueOf(destPtr)
	if v.Elem().Kind() != reflect.Slice || v.Type().Implements(typeOfSQLScanner) {
		return destPtr
	}
	return SliceScanner{destSlice: v.Elem()}
}

// Scan implements the sql.Scanner interface.
func (a *SliceScanner) Scan(src any) error {
	switch src := src.(type) {
	case []byte:
		return a.scanString(string(src))
	case string:
		return a.scanString(src)
	default:
		return fmt.Errorf("can't scan %T as slice", src)
	}
}

func (a *SliceScanner) scanString(src string) error {
	elems, err := nullable.SplitArray(src)
	if err != nil {
		return err
	}
	if len(elems) == 0 {
		a.destSlice.Set(reflect.Zero(a.destSlice.Type()))
		return nil
	}
	elemType := a.destSlice.Type().Elem()
	newSlice := reflect.MakeSlice(elemType, len(elems), len(elems))
	if reflect.PtrTo(elemType).Implements(typeOfSQLScanner) {
		for i, elem := range elems {
			err = newSlice.Index(i).Addr().Interface().(sql.Scanner).Scan(elem)
			if err != nil {
				return fmt.Errorf("can't scan %q as element %d of slice %s because of %w", elem, i, elemType, err)
			}
		}
	} else {
		for i, elem := range elems {
			err = ScanValue(elem, newSlice.Index(i))
			if err != nil {
				return fmt.Errorf("can't scan %q as element %d of slice %s because of %w", elem, i, elemType, err)
			}
		}
	}
	a.destSlice.Set(newSlice)
	return nil
}
