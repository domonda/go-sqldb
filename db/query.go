package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/domonda/go-sqldb"
)

// CurrentTimestamp returns the SQL CURRENT_TIMESTAMP
// for the connection added to the context
// or else the default connection.
//
// Returns time.Now() in case of any error.
//
// Useful for getting the timestamp of a
// SQL transaction for use in Go code.
func CurrentTimestamp(ctx context.Context) time.Time {
	t, err := QueryValue[time.Time](ctx, `SELECT CURRENT_TIMESTAMP`)
	if err != nil {
		return time.Now()
	}
	return t
}

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	conn := Conn(ctx)
	err := conn.Exec(ctx, query, args...)
	if err != nil {
		return wrapErrorWithQuery(err, query, args, conn)
	}
	return nil
}

func ExecStmt(ctx context.Context, query string) (stmtFunc func(ctx context.Context, args ...any) error, closeFunc func() error, err error) {
	conn := Conn(ctx)
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	stmtFunc = func(ctx context.Context, args ...any) error {
		err := stmt.Exec(ctx, args...)
		if err != nil {
			return wrapErrorWithQuery(err, query, args, conn)
		}
		return nil
	}
	return stmtFunc, stmt.Close, nil
}

// QueryRow queries a single row and returns a RowScanner for the results.
func QueryRow(ctx context.Context, query string, args ...any) *RowScanner {
	conn := Conn(ctx)
	rows := conn.Query(ctx, query, args...)
	return NewRowScanner(rows, defaultStructReflector, conn, query, args)
}

// // QueryRows queries multiple rows and returns a RowsScanner for the results.
// func QueryRows(ctx context.Context, query string, args ...any) *MultiRowScanner {
// 	conn := Conn(ctx)
// 	rows := conn.Query(query, args...)
// 	return NewMultiRowScanner(ctx, rows, DefaultStructReflector, conn, query, args)
// }

// QueryValue queries a single value of type T.
func QueryValue[T any](ctx context.Context, query string, args ...any) (value T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		return *new(T), err
	}
	return value, nil
}

// QueryValueReplaceErrNoRows queries a single value of type T.
// In case of an sql.ErrNoRows error, errNoRows will be called
// and its result returned together with the default value for T.
func QueryValueReplaceErrNoRows[T any](ctx context.Context, errNoRows func() error, query string, args ...any) (value T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && errNoRows != nil {
			return *new(T), errNoRows()
		}
		return *new(T), err
	}
	return value, nil
}

// QueryValueOr queries a single value of type T
// or returns the passed defaultValue in case of sql.ErrNoRows.
func QueryValueOr[T any](ctx context.Context, defaultValue T, query string, args ...any) (value T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultValue, nil
		}
		return *new(T), err
	}
	return value, err
}

// QueryRowStruct queries a row and scans it as struct.
func QueryRowStruct[S any](ctx context.Context, query string, args ...any) (row *S, err error) {
	conn := Conn(ctx)
	rows := conn.Query(ctx, query, args...)
	defer rows.Close()
	err = scanStruct(rows, defaultStructReflector, reflect.ValueOf(&row))
	if err != nil {
		return nil, wrapErrorWithQuery(err, query, args, conn)
	}
	return row, nil
}

// QueryRowStructReplaceErrNoRows queries a row and scans it as struct.
// In case of an sql.ErrNoRows error, errNoRows will be called
// and its result returned as error together with nil as row.
func QueryRowStructReplaceErrNoRows[S any](ctx context.Context, errNoRows func() error, query string, args ...any) (row *S, err error) {
	row, err = QueryRowStruct[S](ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && errNoRows != nil {
			return nil, errNoRows()
		}
		return nil, err
	}
	return row, nil
}

// QueryRowStructOrNil queries a row and scans it as struct
// or returns nil in case of sql.ErrNoRows.
func QueryRowStructOrNil[S any](ctx context.Context, query string, args ...any) (row *S, err error) {
	row, err = QueryRowStruct[S](ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetRow uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
func GetRow[S StructWithTableName](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	// Using explicit first pkValue value
	// to not be able to compile without any value
	pkValues = append([]any{pkValue}, pkValues...)
	t := reflect.TypeOf(row).Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct template type instead of %s", t)
	}
	conn := Conn(ctx)
	table, err := defaultStructReflector.TableNameForStruct(t)
	if err != nil {
		return nil, err
	}
	table, err = conn.FormatTableName(table)
	if err != nil {
		return nil, err
	}
	pkColumns, err := pkColumnsOfStruct(defaultStructReflector, t)
	if err != nil {
		return nil, err
	}
	if len(pkColumns) != len(pkValues) {
		return nil, fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	for i, column := range pkColumns {
		pkColumns[i], err = conn.FormatColumnName(column)
		if err != nil {
			return nil, err
		}
	}
	var query strings.Builder
	fmt.Fprintf(&query, `SELECT * FROM %s WHERE %s = %s`, table, pkColumns[0], conn.FormatPlaceholder(0)) //#nosec G104
	for i := 1; i < len(pkColumns); i++ {
		fmt.Fprintf(&query, ` AND %s = %s`, pkColumns[i], conn.FormatPlaceholder(i)) //#nosec G104
	}
	return QueryRowStruct[S](ctx, query.String(), pkValues...)
}

// GetRowOrNil uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValue+pkValues.
func GetRowOrNil[S StructWithTableName](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	row, err = GetRow[S](ctx, pkValue, pkValues...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// QuerySlice returns queried rows as slice of the generic type T
// using the passed reflector to scan column values as struct fields.
// QuerySlice returns queried rows as slice of the generic type T.
func QuerySlice[T any](ctx context.Context, query string, args ...any) (rows []T, err error) {
	sqlRows := Conn(ctx).Query(ctx, query, args...)
	defer sqlRows.Close()

	rows = make([]T, 0, 32)
	sliceVal := reflect.ValueOf(rows)
	sliceElemType := sliceVal.Type().Elem()
	rowStructs := isNonSQLScannerStruct(sliceElemType)

	columns, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	if !rowStructs && len(columns) > 1 {
		return nil, fmt.Errorf("expected single column result for type %s but got %d columns", sliceElemType, len(columns))
	}

	reflector := GetStructReflector(ctx)

	for sqlRows.Next() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		sliceVal = reflect.Append(sliceVal, reflect.Zero(sliceElemType))
		destPtr := sliceVal.Index(sliceVal.Len() - 1).Addr()
		if rowStructs {
			err = scanStruct(sqlRows, reflector, destPtr)
		} else {
			err = sqlRows.Scan(destPtr.Interface())
		}
		if err != nil {
			return nil, err
		}
	}
	if sqlRows.Err() != nil {
		return nil, sqlRows.Err()
	}

	return rows, nil
}

// isNonSQLScannerStruct returns true if the passed type is a struct
// that does not implement the sql.Scanner interface,
// or a pointer to a struct that does not implement the sql.Scanner interface.
func isNonSQLScannerStruct(t reflect.Type) bool {
	if t == typeOfTime || t.Kind() == reflect.Ptr && t.Elem() == typeOfTime {
		return false
	}
	// Struct that does not implement sql.Scanner
	if t.Kind() == reflect.Struct && !reflect.PointerTo(t).Implements(typeOfSQLScanner) {
		return true
	}
	// Pointer to struct that does not implement sql.Scanner
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct && !t.Implements(typeOfSQLScanner) {
		return true
	}
	return false
}

// QueryStrings scans the query result into a table of strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
func QueryStrings(ctx context.Context, query string, args ...any) (rows [][]string, err error) {
	sqlRows := Conn(ctx).Query(ctx, query, args...)
	defer sqlRows.Close()

	cols, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	rows = [][]string{cols}
	stringScannablePtrs := make([]any, len(cols))
	for sqlRows.Next() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		row := make([]string, len(cols))
		// Modify stringScannablePtrs to point to the row values
		for i := range stringScannablePtrs {
			stringScannablePtrs[i] = (*sqldb.StringScannable)(&row[i])
		}
		err := sqlRows.Scan(stringScannablePtrs...)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if sqlRows.Err() != nil {
		return nil, sqlRows.Err()
	}
	return rows, err
}

// QueryCallback calls the passed callback
// with scanned values or a struct for every row.
//
// If the callback function has a single struct or struct pointer argument,
// then RowScanner.ScanStruct will be used per row,
// else RowScanner.Scan will be used for all arguments of the callback.
// If the function has a context.Context as first argument,
// then the passed ctx will be passed on.
//
// The callback can have no result or a single error result value.
//
// If a non nil error is returned from the callback, then this error
// is returned immediately by this function without scanning further rows.
//
// In case of zero rows, no error will be returned.
func QueryCallback(ctx context.Context, callback any, query string, args ...any) error {
	val := reflect.ValueOf(callback)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		return fmt.Errorf("QueryCallback expected callback function, got %s", typ)
	}
	if typ.IsVariadic() {
		return fmt.Errorf("QueryCallback callback function must not be varidic: %s", typ)
	}
	if typ.NumIn() == 0 || (typ.NumIn() == 1 && typ.In(0) == typeOfContext) {
		return fmt.Errorf("QueryCallback callback function has no arguments: %s", typ)
	}
	firstArg := 0
	if typ.In(0) == typeOfContext {
		firstArg = 1
	}
	structArg := isNonSQLScannerStruct(typ.In(firstArg))
	if structArg && typ.NumIn()-firstArg > 1 {
		return fmt.Errorf("QueryCallback callback function must not have further argument after struct: %s", typ)
	}
	for i := firstArg; i < typ.NumIn(); i++ {
		t := typ.In(i)
		if i > firstArg && isNonSQLScannerStruct(t) {
			return fmt.Errorf("QueryCallback callback function argument %d has invalid argument type: %s", i, typ.In(i))
		}
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Chan, reflect.Func:
			return fmt.Errorf("QueryCallback callback function argument %d has invalid argument type: %s", i, typ.In(i))
		}
	}
	if typ.NumOut() > 1 {
		return fmt.Errorf("QueryCallback callback function can only have one result value: %s", typ)
	}
	if typ.NumOut() == 1 && typ.Out(0) != typeOfError {
		return fmt.Errorf("QueryCallback callback function result must be of type error: %s", typ)
	}

	sqlRows := Conn(ctx).Query(ctx, query, args...)
	defer sqlRows.Close()

	if !structArg {
		cols, err := sqlRows.Columns()
		if err != nil {
			return err
		}
		if len(cols) != typ.NumIn()-firstArg {
			return fmt.Errorf("QueryCallback callback function has %d non-context arguments but query result has %d columns", typ.NumIn()-firstArg, len(cols))
		}
	}

	reflector := GetStructReflector(ctx)

	scannedValPtrs := make([]any, typ.NumIn()-firstArg)
	callbackArgs := make([]reflect.Value, typ.NumIn())
	if firstArg == 1 {
		callbackArgs[0] = reflect.ValueOf(ctx)
	}
	for sqlRows.Next() {
		err := ctx.Err()
		if err != nil {
			return err
		}

		// First step is to scan the row
		for i := range scannedValPtrs {
			scannedValPtrs[i] = reflect.New(typ.In(firstArg + i)).Interface()
		}
		if structArg {
			err = scanStruct(sqlRows, reflector, reflect.ValueOf(scannedValPtrs[0]))
		} else {
			err = sqlRows.Scan(scannedValPtrs...)
		}
		if err != nil {
			return err
		}

		// Then do callback using reflection
		for i := firstArg; i < len(args); i++ {
			callbackArgs[i] = reflect.ValueOf(scannedValPtrs[i-firstArg]).Elem()
		}
		res := val.Call(callbackArgs)
		if len(res) > 0 && !res[0].IsNil() {
			return res[0].Interface().(error)
		}
	}

	return sqlRows.Err()
}
