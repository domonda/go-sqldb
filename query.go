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

func ContextConnection(ctx context.Context) Connection {
	panic("TODO")
}

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

// Query queries multiple rows and returns a RowsScanner for the results.
func Query(ctx context.Context, query string, args ...any) RowsScanner {
	conn := ContextConnection(ctx)
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		rows = RowsWithError(err)
	}
	return NewRowsScanner(ctx, rows, query, args, conn)
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

// QueryStructSlice returns queried rows as slice of the generic type S
// which must be a struct or a pointer to a struct.
func QueryStructSlice[S any](ctx context.Context, query string, args ...any) (rows []S, err error) {
	err = Query(ctx, query, args...).ScanStructSlice(&rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
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

	val := reflect.ValueOf(callback)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		return fmt.Errorf("expected callback function, got %s", typ)
	}
	if typ.IsVariadic() {
		return fmt.Errorf("callback function must not be varidic: %s", typ)
	}
	if typ.NumIn() == 0 || (typ.NumIn() == 1 && typ.In(0) == typeOfContext) {
		return fmt.Errorf("callback function has no arguments: %s", typ)
	}
	firstArg := 0
	if typ.In(0) == typeOfContext {
		firstArg = 1
	}
	structArg := false
	for i := firstArg; i < typ.NumIn(); i++ {
		t := typ.In(i)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t == typeOfTime {
			continue
		}
		switch t.Kind() {
		case reflect.Struct:
			if t.Implements(typeOfSQLScanner) || reflect.PtrTo(t).Implements(typeOfSQLScanner) {
				continue
			}
			if structArg {
				return fmt.Errorf("callback function must not have further argument after struct: %s", typ)
			}
			structArg = true
		case reflect.Chan, reflect.Func:
			return fmt.Errorf("callback function has invalid argument type: %s", typ.In(i))
		}
	}
	if typ.NumOut() > 1 {
		return fmt.Errorf("callback function can only have one result value: %s", typ)
	}
	if typ.NumOut() == 1 && typ.Out(0) != typeOfError {
		return fmt.Errorf("callback function result must be of type error: %s", typ)
	}

	f := func(row RowScanner) (err error) {
		// First scan row
		scannedValPtrs := make([]any, typ.NumIn()-firstArg)
		for i := range scannedValPtrs {
			scannedValPtrs[i] = reflect.New(typ.In(firstArg + i)).Interface()
		}
		if structArg {
			err = row.ScanStruct(scannedValPtrs[0])
		} else {
			err = row.Scan(scannedValPtrs...)
		}
		if err != nil {
			return err
		}

		// Then do callback via reflection
		args := make([]reflect.Value, typ.NumIn())
		if firstArg == 1 {
			args[0] = reflect.ValueOf(ctx)
		}
		for i := firstArg; i < len(args); i++ {
			args[i] = reflect.ValueOf(scannedValPtrs[i-firstArg]).Elem()
		}
		res := val.Call(args)
		if len(res) > 0 && !res[0].IsNil() {
			return res[0].Interface().(error)
		}
		return nil
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
		err := f(NewCurrentRowScanner(rows, query, args, conn))
		if err != nil {
			return err
		}
	}
	return rows.Err()
}
