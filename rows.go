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

func (s *rowsScanner) Columns() ([]string, error) {
	return s.rows.Columns()
}

func (s *rowsScanner) ScanSlice(dest any) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
	if err != nil {
		return WrapErrorWithQuery(err, s.query, s.args, s.conn)
	}
	return nil
}

func (s *rowsScanner) ScanStructSlice(dest any) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, s.conn)
	if err != nil {
		return WrapErrorWithQuery(err, s.query, s.args, s.conn)
	}
	return nil
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
		err = errors.Join(err, s.rows.Close())
		err = WrapErrorWithQuery(err, s.query, s.args, s.conn)
	}()

	panic("TODO")
	// for s.rows.Next() {
	// 	if s.ctx.Err() != nil {
	// 		return s.ctx.Err()
	// 	}

	// 	err := callback(CurrentRowScanner{s.rows, s.structFieldMapper})
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// return s.rows.Err()
}

func (s *rowsScanner) ForEachRowCall(callback any) error {
	panic("TODO")
	// 	val := reflect.ValueOf(callback)
	// 	typ := val.Type()
	// 	if typ.Kind() != reflect.Func {
	// 		return fmt.Errorf("ForEachRowCall expected callback function, got %s", typ)
	// 	}
	// 	if typ.IsVariadic() {
	// 		return fmt.Errorf("ForEachRowCall callback function must not be varidic: %s", typ)
	// 	}
	// 	if typ.NumIn() == 0 || (typ.NumIn() == 1 && typ.In(0) == typeOfContext) {
	// 		return fmt.Errorf("ForEachRowCall callback function has no arguments: %s", typ)
	// 	}
	// 	firstArg := 0
	// 	if typ.In(0) == typeOfContext {
	// 		firstArg = 1
	// 	}
	// 	structArg := false
	// 	for i := firstArg; i < typ.NumIn(); i++ {
	// 		t := typ.In(i)
	// 		for t.Kind() == reflect.Ptr {
	// 			t = t.Elem()
	// 		}
	// 		if t == typeOfTime {
	// 			continue
	// 		}
	// 		switch t.Kind() {
	// 		case reflect.Struct:
	// 			if t.Implements(typeOfSQLScanner) || reflect.PtrTo(t).Implements(typeOfSQLScanner) {
	// 				continue
	// 			}
	// 			if structArg {
	// 				return fmt.Errorf("ForEachRowCall callback function must not have further argument after struct: %s", typ)
	// 			}
	// 			structArg = true
	// 		case reflect.Chan, reflect.Func:
	// 			return fmt.Errorf("ForEachRowCall callback function has invalid argument type: %s", typ.In(i))
	// 		}
	// 	}
	// 	if typ.NumOut() > 1 {
	// 		return fmt.Errorf("ForEachRowCall callback function can only have one result value: %s", typ)
	// 	}
	// 	if typ.NumOut() == 1 && typ.Out(0) != typeOfError {
	// 		return fmt.Errorf("ForEachRowCall callback function result must be of type error: %s", typ)
	// 	}

	// 	f = func(row RowScanner) (err error) {
	// 		// First scan row
	// 		scannedValPtrs := make([]any, typ.NumIn()-firstArg)
	// 		for i := range scannedValPtrs {
	// 			scannedValPtrs[i] = reflect.New(typ.In(firstArg + i)).Interface()
	// 		}
	// 		if structArg {
	// 			err = row.ScanStruct(scannedValPtrs[0])
	// 		} else {
	// 			err = row.Scan(scannedValPtrs...)
	// 		}
	// 		if err != nil {
	// 			return err
	// 		}

	//		// Then do callback via reflection
	//		args := make([]reflect.Value, typ.NumIn())
	//		if firstArg == 1 {
	//			args[0] = reflect.ValueOf(s.ctx)
	//		}
	//		for i := firstArg; i < len(args); i++ {
	//			args[i] = reflect.ValueOf(scannedValPtrs[i-firstArg]).Elem()
	//		}
	//		res := val.Call(args)
	//		if len(res) > 0 && !res[0].IsNil() {
	//			return res[0].Interface().(error)
	//		}
	//		return nil
	//	}
	//
	// return f, nil
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
