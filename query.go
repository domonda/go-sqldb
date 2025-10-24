package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// QueryRow queries a single row and returns a Row for the results.
func QueryRow(ctx context.Context, conn Querier, reflector StructReflector, query string, args ...any) *Row {
	rows := conn.Query(ctx, query, args...)
	return NewRow(rows, reflector, GetQueryFormatter(conn), query, args)
}

// QueryValue queries a single value mapped to the type T.
func QueryValue[T any](ctx context.Context, conn Querier, reflector StructReflector, query string, args ...any) (val T, err error) {
	err = QueryRow(ctx, conn, reflector, query, args...).Scan(&val)
	if err != nil {
		return *new(T), err
	}
	return val, nil
}

// QueryValueOr queries a single value of type T
// or returns the passed defaultVal in case of sql.ErrNoRows.
func QueryValueOr[T any](ctx context.Context, conn Querier, reflector StructReflector, defaultVal T, query string, args ...any) (val T, err error) {
	val, err = QueryValue[T](ctx, conn, reflector, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultVal, nil
	}
	return val, err
}

func QueryValueStmt[T any](ctx context.Context, conn Preparer, reflector StructReflector, query string) (queryFunc func(ctx context.Context, args ...any) (T, error), closeStmt func() error, err error) {
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		err = fmt.Errorf("can't prepare query because: %w", err)
		return nil, nil, WrapErrorWithQuery(err, query, nil, GetQueryFormatter(conn))
	}

	queryFunc = func(ctx context.Context, args ...any) (val T, err error) {
		rows := stmt.Query(ctx, args...)
		err = NewRow(rows, reflector, GetQueryFormatter(conn), query, args).Scan(&val)
		return val, err
	}
	return queryFunc, stmt.Close, nil
}

// ReadRowStructWithTableName uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type `*S` that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
func ReadRowStructWithTableName[S StructWithTableName](ctx context.Context, conn Querier, queryBuilder QueryBuilder, reflector StructReflector, pkValue any, pkValues ...any) (S, error) {
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
	table, err := reflector.TableNameForStruct(t)
	if err != nil {
		return *new(S), err
	}
	table, err = queryBuilder.FormatTableName(table)
	if err != nil {
		return *new(S), err
	}
	pkColumns, err := PrimaryKeyColumnsOfStruct(reflector, t)
	if err != nil {
		return *new(S), err
	}
	if len(pkColumns) != len(pkValues) {
		return *new(S), fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	for i, column := range pkColumns {
		pkColumns[i], err = queryBuilder.FormatColumnName(column)
		if err != nil {
			return *new(S), err
		}
	}
	query, err := queryBuilder.QueryRowWithPK(table, pkColumns)
	if err != nil {
		return *new(S), err
	}
	return QueryValue[S](ctx, conn, reflector, query, pkValues...)
}

// ReadRowStructWithTableNameOr uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValue+pkValues.
func ReadRowStructWithTableNameOr[S StructWithTableName](ctx context.Context, conn Querier, queryBuilder QueryBuilder, reflector StructReflector, defaultVal S, pkValue any, pkValues ...any) (S, error) {
	row, err := ReadRowStructWithTableName[S](ctx, conn, queryBuilder, reflector, pkValue, pkValues...)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultVal, nil
	}
	return row, err
}

// QueryRowAsMap queries a single row and returns the columns as map
// using the column names as keys.
func QueryRowAsMap[K ~string, V any](ctx context.Context, conn Querier, queryFmt QueryFormatter, query string, args ...any) (m map[K]V, err error) {
	rows := conn.Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, rows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, queryFmt)
		}
	}()

	// Check if there was an error even before preparing the row with Next()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if !rows.Next() {
		// Error during preparing the row with Next()
		if rows.Err() != nil {
			return nil, rows.Err()
		}
		return nil, sql.ErrNoRows
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	vals := make([]V, len(columns))
	valPtrs := make([]any, len(columns))
	for i := range valPtrs {
		valPtrs[i] = &vals[i]
	}
	err = rows.Scan(valPtrs...)
	if err != nil {
		return nil, err
	}

	m = make(map[K]V, len(columns))
	for i, column := range columns {
		m[K(column)] = vals[i]
	}
	return m, nil
}

// QueryRowsAsSlice returns queried rows as slice of the generic type T
// using the passed reflector to scan column values as struct fields.
// QueryRowsAsSlice returns queried rows as slice of the generic type T.
func QueryRowsAsSlice[T any](ctx context.Context, conn Querier, reflector StructReflector, query string, args ...any) (rows []T, err error) {
	sqlRows := conn.Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, sqlRows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, GetQueryFormatter(conn))
		}
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

// QueryRowsAsStrings scans the query result into a table of strings
// where the first row is a header row with the column names.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
//
// If the query result has no rows, then only the header row
// and no error will be returned.
func QueryRowsAsStrings(ctx context.Context, conn Querier, query string, args ...any) (rows [][]string, err error) {
	sqlRows := conn.Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, sqlRows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, GetQueryFormatter(conn))
		}
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
			stringScannablePtrs[i] = (*StringScannable)(&row[i])
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
func QueryCallback(ctx context.Context, conn Querier, reflector StructReflector, callback any, query string, args ...any) (err error) {
	defer func() {
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, GetQueryFormatter(conn))
		}
	}()
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
		for t.Kind() == reflect.Pointer {
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

	sqlRows := conn.Query(ctx, query, args...)
	defer sqlRows.Close()

	columns, err := sqlRows.Columns()
	if err != nil {
		return err
	}
	if !structArg && len(columns) != typ.NumIn()-firstArg {
		return fmt.Errorf("QueryCallback callback function has %d non-context arguments but query result has %d columns", typ.NumIn()-firstArg, len(columns))
	}

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
