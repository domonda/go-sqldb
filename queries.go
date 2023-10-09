package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Now returns the result of the SQL NOW()
// function for the current connection.
// Useful for getting the timestamp of a
// SQL transaction for use in Go code.
func Now(ctx context.Context) (time.Time, error) {
	return QueryValue[time.Time](ctx, `SELECT NOW()`)
}

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	conn := ContextConnection(ctx)
	err := conn.Exec(ctx, query, args...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, conn)
	}
	return nil
}

// QueryRow queries a single row and returns a RowScanner for the results.
func QueryRow(ctx context.Context, query string, args ...any) RowScanner {
	conn := ContextConnection(ctx)
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		rows = RowsWithError(err)
	}
	return NewRowScanner(rows, query, args, conn)
}

// QueryValue queries a single value of type T.
func QueryValue[T any](ctx context.Context, query string, args ...any) (value T, err error) {
	err = QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		var zero T
		return zero, err
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
		var zero T
		return zero, err
	}
	return value, err
}

// QueryRowStruct queries a row and scans it as struct.
func QueryRowStruct[S any](ctx context.Context, query string, args ...any) (row *S, err error) {
	err = QueryRow(ctx, query, args...).ScanStruct(&row)
	if err != nil {
		return nil, err
	}
	return row, nil
}

// QueryRowStructOrNil queries a row and scans it as struct
// or returns nil in case of sql.ErrNoRows.
func QueryRowStructOrNil[S any](ctx context.Context, query string, args ...any) (row *S, err error) {
	err = QueryRow(ctx, query, args...).ScanStruct(&row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetRowStruct uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
func GetRowStruct[S any](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	// Using explicit first pkValue value
	// to not be able to compile without any value
	pkValues = append([]any{pkValue}, pkValues...)
	t := reflect.TypeOf(row).Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct template type instead of %s", t)
	}
	conn := ContextConnection(ctx)
	table, pkColumns, err := pkColumnsOfStruct(conn, t)
	if err != nil {
		return nil, err
	}
	if len(pkColumns) != len(pkValues) {
		return nil, fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	var query strings.Builder
	fmt.Fprintf(&query, `SELECT * FROM %s WHERE "%s" = $1`, table, pkColumns[0]) //#nosec G104
	for i := 1; i < len(pkColumns); i++ {
		fmt.Fprintf(&query, ` AND "%s" = $%d`, pkColumns[i], i+1) //#nosec G104
	}
	return QueryRowStruct[S](ctx, query.String(), pkValues...)
}

// GetRowStructOrNil uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValue+pkValues.
func GetRowStructOrNil[S any](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	row, err = GetRowStruct[S](ctx, pkValue, pkValues...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// QueryRowsAsSlice scans one value per row into one slice element of rowVals.
// dest must be a pointer to a slice with a row value compatible element type. TODO
//
// In case of a cancelled context the rows scanned before the cancellation
// will be returned together with the context error.
func QueryRowsAsSlice[T any](ctx context.Context, query string, args ...any) (rows []T, err error) {
	conn := ContextConnection(ctx)
	defer WrapResultErrorWithQuery(&err, query, args, conn)

	srcRows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var elem T
	scanningStructs := isStructRowType(reflect.TypeOf(elem))
	for srcRows.Next() {
		if ctx.Err() != nil {
			return rows, ctx.Err()
		}
		if scanningStructs {
			err = ScanStruct(srcRows, &elem, conn)
		} else {
			err = srcRows.Scan(&elem)
		}
		if err != nil {
			return rows, err
		}
		rows = append(rows, elem)
	}
	return rows, srcRows.Err()
}

func isStructRowType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return false
	}
	if t.Implements(typeOfSQLScanner) {
		return false
	}
	if reflect.PointerTo(t).Implements(typeOfSQLScanner) {
		return false
	}
	return true
}

// QueryWithRowCallback will call the passed callback function with scanned values
// or a struct for every row.
// If the callback function has a single struct or struct pointer argument,
// then RowScanner.ScanStruct will be used per row,
// else RowScanner.Scan will be used for all arguments of the callback.
// If the function has a context.Context as first argument,
// then the context of the query call will be passed on.
// The callback can have no result or a single error result value.
// If a non nil error is returned from the callback, then this error
// is returned immediately by this function without scanning further rows.
// In case of zero rows, no error will be returned.
func QueryWithRowCallback[F any](ctx context.Context, callback F, query string, args ...any) (err error) {
	conn := ContextConnection(ctx)
	defer WrapResultErrorWithQuery(&err, query, args, conn)

	funcVal := reflect.ValueOf(callback)
	funcType := funcVal.Type()
	if funcType.Kind() != reflect.Func {
		return fmt.Errorf("expected callback function, got %s", funcType)
	}
	if funcType.IsVariadic() {
		return fmt.Errorf("callback function must not be varidic: %s", funcType)
	}
	if funcType.NumIn() == 0 || (funcType.NumIn() == 1 && funcType.In(0) == typeOfContext) {
		return fmt.Errorf("callback function has no arguments: %s", funcType)
	}
	firstArgIndex := 0
	hasCtxArg := false
	if funcType.In(0) == typeOfContext {
		hasCtxArg = true
		firstArgIndex = 1
	}
	hasStructArg := false
	for i := firstArgIndex; i < funcType.NumIn(); i++ {
		t := funcType.In(i)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Struct:
			if t.Implements(typeOfSQLScanner) || reflect.PointerTo(t).Implements(typeOfSQLScanner) {
				continue
			}
			if hasStructArg {
				return fmt.Errorf("callback function must not have further argument after struct: %s", funcType)
			}
			hasStructArg = true
		case reflect.Chan, reflect.Func:
			return fmt.Errorf("callback function has invalid argument type: %s", funcType.In(i))
		}
	}
	if funcType.NumOut() == 1 && funcType.Out(0) != typeOfError {
		return fmt.Errorf("callback function result must be of type error: %s", funcType)
	}
	if funcType.NumOut() > 1 {
		return fmt.Errorf("callback function can only have one result value: %s", funcType)
	}

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// First scan row
		scannedValPtrs := make([]any, funcType.NumIn()-firstArgIndex)
		for i := range scannedValPtrs {
			scannedValPtrs[i] = reflect.New(funcType.In(firstArgIndex + i)).Interface()
		}
		if hasStructArg {
			err = ScanStruct(rows, scannedValPtrs[0], conn)
		} else {
			err = rows.Scan(scannedValPtrs...)
		}
		if err != nil {
			return err
		}

		// Then do callback via reflection
		args := make([]reflect.Value, funcType.NumIn())
		if hasCtxArg {
			args[0] = reflect.ValueOf(ctx)
		}
		for i := firstArgIndex; i < len(args); i++ {
			args[i] = reflect.ValueOf(scannedValPtrs[i-firstArgIndex]).Elem()
		}
		result := funcVal.Call(args)
		if len(result) > 0 && !result[0].IsNil() {
			return result[0].Interface().(error)
		}
	}
	return rows.Err()
}

// QueryStrings returns the queried row values as strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint.
// The first row is a header with the column names.
func QueryStrings(ctx context.Context, query string, args ...any) (rows [][]string, err error) {
	conn := ContextConnection(ctx)
	defer WrapResultErrorWithQuery(&err, query, args, conn)

	srcRows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	cols, err := srcRows.Columns()
	if err != nil {
		return nil, err
	}
	rows = [][]string{cols}
	stringScannablePtrs := make([]any, len(cols))
	for srcRows.Next() {
		if ctx.Err() != nil {
			return rows, ctx.Err()
		}
		row := make([]string, len(cols))
		for i := range stringScannablePtrs {
			stringScannablePtrs[i] = (*StringScannable)(&row[i])
		}
		err := srcRows.Scan(stringScannablePtrs...)
		if err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}
	return rows, srcRows.Err()
}

func Insert(ctx context.Context, table string, rows any) error {
	panic("TODO")
}

func InsertRow(ctx context.Context, row RowWithTableName) error {
	panic("TODO")
}

func InsertRows[R RowWithTableName](ctx context.Context, rows []RowWithTableName) error {
	panic("TODO")
}

func writeInsertQuery(w *strings.Builder, table string, names []string, formatter QueryFormatter) {
	fmt.Fprintf(w, `INSERT INTO %s(`, table)
	for i, name := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteByte('"')
		w.WriteString(name)
		w.WriteByte('"')
	}
	w.WriteString(`) VALUES(`)
	for i := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteString(formatter.ColumnPlaceholder(i))
	}
	w.WriteByte(')')
}
