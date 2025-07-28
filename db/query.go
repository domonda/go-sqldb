package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
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
	t, err := QueryRowValue[time.Time](ctx, `SELECT CURRENT_TIMESTAMP`)
	if err != nil {
		return time.Now()
	}
	return t
}

// QueryRow queries a single row and returns a RowScanner for the results.
func QueryRow(ctx context.Context, query string, args ...any) *Row {
	conn := Conn(ctx)
	rows := conn.Query(ctx, query, args...)
	return NewRow(rows, conn, query, args)
}

// QueryRowValue queries a single row mapped to the type T.
func QueryRowValue[T any](ctx context.Context, query string, args ...any) (val T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&val)
	if err != nil {
		return *new(T), err
	}
	return val, nil
}

// QueryRowValueOr queries a single value of type T
// or returns the passed defaultVal in case of sql.ErrNoRows.
func QueryRowValueOr[T any](ctx context.Context, defaultVal T, query string, args ...any) (val T, err error) {
	val, err = QueryRowValue[T](ctx, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultVal, nil
	}
	return val, err
}

func QueryRowValueStmt[T any](ctx context.Context, query string) (queryFunc func(ctx context.Context, args ...any) (T, error), closeStmt func() error, err error) {
	conn := Conn(ctx)
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		err = fmt.Errorf("can't prepare query because: %w", err)
		return nil, nil, sqldb.WrapErrorWithQuery(err, query, nil, conn)
	}

	queryFunc = func(ctx context.Context, args ...any) (val T, err error) {
		rows := stmt.Query(ctx, args...)
		err = NewRow(rows, conn, query, args).Scan(&val)
		return val, err
	}
	return queryFunc, stmt.Close, nil
}

// ReadRowStruct uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type `*S` that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
func ReadRowStruct[S sqldb.StructWithTableName](ctx context.Context, pkValue any, pkValues ...any) (S, error) {
	// Using explicit first pkValue value
	// to not be able to compile without any value
	pkValues = append([]any{pkValue}, pkValues...)
	t := reflect.TypeFor[S]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return *new(S), fmt.Errorf("expected struct or pointer to struct, got %s", reflect.TypeFor[S]())
	}
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)
	table, err := defaultStructReflector.TableNameForStruct(t)
	if err != nil {
		return *new(S), err
	}
	table, err = conn.FormatTableName(table)
	if err != nil {
		return *new(S), err
	}
	pkColumns, err := pkColumnsOfStruct(defaultStructReflector, t)
	if err != nil {
		return *new(S), err
	}
	if len(pkColumns) != len(pkValues) {
		return *new(S), fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	for i, column := range pkColumns {
		pkColumns[i], err = conn.FormatColumnName(column)
		if err != nil {
			return *new(S), err
		}
	}

	query, err := queryBuilder.QueryRowWithPK(table, pkColumns)
	if err != nil {
		return *new(S), err
	}
	return QueryRowValue[S](ctx, query, pkValues...)
}

// ReadRowStructOr uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValue+pkValues.
func ReadRowStructOr[S sqldb.StructWithTableName](ctx context.Context, defaultVal S, pkValue any, pkValues ...any) (S, error) {
	row, err := ReadRowStruct[S](ctx, pkValue, pkValues...)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultVal, nil
	}
	return row, err
}

// QueryRowsAsSlice returns queried rows as slice of the generic type T
// using the passed reflector to scan column values as struct fields.
// QueryRowsAsSlice returns queried rows as slice of the generic type T.
func QueryRowsAsSlice[T any](ctx context.Context, query string, args ...any) (rows []T, err error) {
	sqlRows := Conn(ctx).Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, sqlRows.Close())
	}()

	sliceElemType := reflect.TypeOf(rows).Elem()
	rowStructs := isNonSQLScannerStruct(sliceElemType)

	columns, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	if !rowStructs && len(columns) > 1 {
		return nil, fmt.Errorf("expected single column result for type %s but got %d columns", sliceElemType, len(columns))
	}

	var reflector sqldb.StructReflector
	if rowStructs {
		reflector = GetStructReflector(ctx)
	}

	for sqlRows.Next() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		rows = append(rows, *new(T))
		if rowStructs {
			err = scanStruct(sqlRows, columns, reflector, &rows[len(rows)-1])
		} else {
			err = sqlRows.Scan(&rows[len(rows)-1])
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

// QueryRowsAsStrings scans the query result into a table of strings
// where the first row is a header row with the column names.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
//
// If the query result has no rows, then only the header row
// and no error will be returned.
func QueryRowsAsStrings(ctx context.Context, query string, args ...any) (rows [][]string, err error) {
	sqlRows := Conn(ctx).Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, sqlRows.Close())
	}()

	cols, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	rows = [][]string{cols}
	stringScannablePtrs := make([]any, len(cols))
	for sqlRows.Next() {
		if err = ctx.Err(); err != nil {
			return nil, err
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
	if err = sqlRows.Err(); err != nil {
		return nil, err
	}
	return rows, nil
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

	columns, err := sqlRows.Columns()
	if err != nil {
		return err
	}
	if !structArg && len(columns) != typ.NumIn()-firstArg {
		return fmt.Errorf("QueryCallback callback function has %d non-context arguments but query result has %d columns", typ.NumIn()-firstArg, len(columns))
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
			err = scanStruct(sqlRows, columns, reflector, scannedValPtrs[0])
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
