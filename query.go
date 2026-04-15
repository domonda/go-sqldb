package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
)

// UnlimitedMaxNumRows is the sentinel value for the maxNumRows argument
// of [QueryRowsAsSlice], [QueryRowsAsStrings] and [QueryRowsAsMapSlice]
// that disables the row cap. Any negative integer has the same effect,
// but using this named constant makes the intent explicit at call sites.
const UnlimitedMaxNumRows = -1

// QueryRow queries a single row and returns a Row for the results.
func QueryRow(ctx context.Context, conn Querier, refl StructReflector, fmtr QueryFormatter, query string, args ...any) *Row {
	rows := conn.Query(ctx, query, args...)
	return NewRow(rows, refl, fmtr, query, args)
}

// QueryRowAs queries a single row and scans it as the type T.
// If T is a struct that does not implement sql.Scanner,
// the column values are scanned into the struct fields.
func QueryRowAs[T any](ctx context.Context, conn Querier, refl StructReflector, fmtr QueryFormatter, query string, args ...any) (val T, err error) {
	err = QueryRow(ctx, conn, refl, fmtr, query, args...).Scan(&val)
	if err != nil {
		return *new(T), err
	}
	return val, nil
}

// QueryRowAsOr queries a single row and scans it as the type T,
// or returns the passed defaultVal in case of sql.ErrNoRows.
func QueryRowAsOr[T any](ctx context.Context, conn Querier, refl StructReflector, fmtr QueryFormatter, defaultVal T, query string, args ...any) (val T, err error) {
	val, err = QueryRowAs[T](ctx, conn, refl, fmtr, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultVal, nil
	}
	return val, err
}

// QueryRowAsStmt prepares the query and returns a function that
// executes it with arguments and scans a single row as the type T.
// The returned closeStmt function must be called to release the prepared statement.
func QueryRowAsStmt[T any](ctx context.Context, conn Preparer, refl StructReflector, fmtr QueryFormatter, query string) (queryFunc func(ctx context.Context, args ...any) (T, error), closeStmt func() error, err error) {
	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		err = fmt.Errorf("failed to prepare query: %w", err)
		return nil, nil, WrapErrorWithQuery(err, query, nil, fmtr)
	}

	queryFunc = func(ctx context.Context, args ...any) (val T, err error) {
		rows := stmt.Query(ctx, args...)
		err = NewRow(rows, refl, fmtr, query, args).Scan(&val)
		return val, err
	}
	return queryFunc, stmt.Close, nil
}

// QueryRowStruct queries a table row by primary key and scans it into a struct of type S.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Primary key columns are identified by fields with the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The number of pkValue+pkValues must match the number of primary key columns.
func QueryRowStruct[S StructWithTableName](ctx context.Context, conn Querier, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, pkValue any, pkValues ...any) (S, error) {
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

	// Try cache lookup
	queryRowStructCacheMtx.RLock()
	cached, ok := queryRowStructCache[t][refl][builder][fmtr]
	queryRowStructCacheMtx.RUnlock()
	if ok {
		if cached.numPKColumns != len(pkValues) {
			return *new(S), fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, cached.numPKColumns)
		}
		return QueryRowAs[S](ctx, conn, refl, fmtr, cached.query, pkValues...)
	}

	// Cache miss — build query
	table, err := refl.TableNameForStruct(t)
	if err != nil {
		return *new(S), err
	}
	pkColumns, err := refl.PrimaryKeyColumnsOfStruct(t)
	if err != nil {
		return *new(S), err
	}
	if len(pkColumns) == 0 {
		return *new(S), fmt.Errorf("QueryRowStruct of table %s: %s has no primary key fields", table, t)
	}
	if len(pkColumns) != len(pkValues) {
		return *new(S), fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	query, err := builder.QueryRowWithPK(fmtr, table, pkColumns)
	if err != nil {
		return *new(S), err
	}

	// Store in cache
	queryRowStructCacheMtx.Lock()
	if _, ok := queryRowStructCache[t]; !ok {
		queryRowStructCache[t] = make(map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryRowStructCacheEntry)
	}
	if _, ok := queryRowStructCache[t][refl]; !ok {
		queryRowStructCache[t][refl] = make(map[QueryBuilder]map[QueryFormatter]queryRowStructCacheEntry)
	}
	if _, ok := queryRowStructCache[t][refl][builder]; !ok {
		queryRowStructCache[t][refl][builder] = make(map[QueryFormatter]queryRowStructCacheEntry)
	}
	queryRowStructCache[t][refl][builder][fmtr] = queryRowStructCacheEntry{
		query:        query,
		numPKColumns: len(pkColumns),
	}
	queryRowStructCacheMtx.Unlock()

	return QueryRowAs[S](ctx, conn, refl, fmtr, query, pkValues...)
}

// QueryRowStructOr queries a table row by primary key and scans it into a struct of type S.
// Returns defaultVal and no error if no row was found.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Primary key columns are identified by fields with the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The number of pkValue+pkValues must match the number of primary key columns.
func QueryRowStructOr[S StructWithTableName](ctx context.Context, conn Querier, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, defaultVal S, pkValue any, pkValues ...any) (S, error) {
	row, err := QueryRowStruct[S](ctx, conn, refl, builder, fmtr, pkValue, pkValues...)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultVal, nil
	}
	return row, err
}

// QueryRowAsMap queries a single row and returns the columns as map
// using the column names as keys.
func QueryRowAsMap[K ~string, V any](ctx context.Context, conn Querier, fmtr QueryFormatter, query string, args ...any) (m map[K]V, err error) {
	rows := conn.Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, rows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, fmtr)
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

// QueryRowAsStrings queries a single row and returns its column values as strings.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
func QueryRowAsStrings(ctx context.Context, conn Querier, fmtr QueryFormatter, query string, args ...any) ([]string, error) {
	return QueryRow(ctx, conn, nil, fmtr, query, args...).ScanStrings()
}

// QueryRowAsStringsWithHeader queries a single row and returns a [][]string
// where the first slice contains the column names and the second slice
// contains the row values as strings.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
func QueryRowAsStringsWithHeader(ctx context.Context, conn Querier, fmtr QueryFormatter, query string, args ...any) ([][]string, error) {
	row := QueryRow(ctx, conn, nil, fmtr, query, args...)
	cols, err := row.Columns()
	if err != nil {
		return nil, err
	}
	vals, err := row.ScanStrings()
	if err != nil {
		return nil, err
	}
	return [][]string{cols, vals}, nil
}

// QueryRowsAsMapSlice queries rows and returns them as a slice of maps
// keyed by column name. The values are exactly how they are passed
// from the database driver to an [sql.Scanner]. Byte slices will be copied.
//
// If converter is not nil, it is applied to each scanned value and
// replaces the value in the returned map when it reports a successful
// conversion. Multiple converters can be combined by passing a
// [ScanConverters] slice.
//
// Pass [UnlimitedMaxNumRows] (or any negative integer) for maxNumRows
// to disable the limit. A value of 0 is enforced as a hard cap that
// permits no rows: an empty query returns no rows and no error, a
// non-empty query returns no rows together with [ErrMaxNumRowsExceeded].
// Non-negative values cap the number of rows; exceeding the cap returns
// [ErrMaxNumRowsExceeded] along with the rows scanned so far.
//
// On any error (context cancellation, scan failure, or the final rows.Err()),
// the function returns whatever was scanned before the error together with
// the wrapped error, so callers may still consume the partial result.
//
// If a row contains duplicate column names,
// later columns overwrite earlier ones in its map.
//
// Use this as the multi-row counterpart of [Row.ScanMap],
// for example to encode a query result as a JSON array.
func QueryRowsAsMapSlice(ctx context.Context, conn Querier, fmtr QueryFormatter, converter ScanConverter, maxNumRows int, query string, args ...any) (result []map[string]any, err error) {
	sqlRows := conn.Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, sqlRows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, fmtr)
		}
	}()

	if maxNumRows < 0 {
		maxNumRows = math.MaxInt // Practically unlimited
	}

	cols, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	var (
		anys     = make([]AnyValue, len(cols))
		scanPtrs = make([]any, len(cols))
	)
	for i := range scanPtrs {
		scanPtrs[i] = &anys[i]
	}
	for sqlRows.Next() {
		if len(result) >= maxNumRows {
			return result, ErrMaxNumRowsExceeded{MaxNumRows: maxNumRows}
		}
		if err = ctx.Err(); err != nil {
			return result, err
		}
		if err = sqlRows.Scan(scanPtrs...); err != nil {
			return result, err
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			v := anys[i].Val
			if converter != nil {
				if cv, ok := converter.ConvertValue(v); ok {
					v = cv
				}
			}
			row[col] = v
		}
		result = append(result, row)
	}
	if err = sqlRows.Err(); err != nil {
		return result, err
	}
	return result, nil
}

// QueryRowsAsSlice returns queried rows as slice of the generic type T
// using the passed reflector to scan column values as struct fields.
//
// Pass [UnlimitedMaxNumRows] (or any negative integer) for maxNumRows
// to disable the limit. A value of 0 is enforced as a hard cap that
// permits no rows: an empty query returns no rows and no error, a
// non-empty query returns no rows together with [ErrMaxNumRowsExceeded].
// Non-negative values cap the number of rows; exceeding the cap returns
// [ErrMaxNumRowsExceeded] along with the rows scanned so far.
//
// On any error (context cancellation, scan failure, or the final rows.Err()),
// the function returns whatever was scanned before the error together with
// the wrapped error, so callers may still consume the partial result.
func QueryRowsAsSlice[T any](ctx context.Context, conn Querier, refl StructReflector, fmtr QueryFormatter, maxNumRows int, query string, args ...any) (rows []T, err error) {
	sqlRows := conn.Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, sqlRows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, fmtr)
		}
	}()

	if maxNumRows < 0 {
		maxNumRows = math.MaxInt // Practically unlimited
	}

	sliceElemType := reflect.TypeFor[T]()
	rowStructs := isNonSQLScannerStruct(sliceElemType)

	columns, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	if !rowStructs && len(columns) > 1 {
		return nil, fmt.Errorf("expected single column result for type %s but got %d columns", sliceElemType, len(columns))
	}

	for sqlRows.Next() {
		if len(rows) >= maxNumRows {
			return rows, ErrMaxNumRowsExceeded{MaxNumRows: maxNumRows}
		}
		if err = ctx.Err(); err != nil {
			return rows, err
		}
		rows = append(rows, *new(T))
		if rowStructs {
			err = scanStruct(sqlRows, columns, refl, &rows[len(rows)-1])
		} else {
			err = sqlRows.Scan(&rows[len(rows)-1])
		}
		if err != nil {
			// Drop the row we appended for this iteration; Scan may have
			// partially filled it before failing, and callers consuming the
			// partial result should not see a half-scanned tail.
			rows = rows[:len(rows)-1]
			return rows, err
		}
	}
	if err = sqlRows.Err(); err != nil {
		return rows, err
	}
	return rows, nil
}

// QueryRowsAsStrings scans the query result into a table of strings
// where the first row is a header row with the column names.
// The returned [][]string therefore always has rows[0] set to the column
// header when [Rows.Columns] succeeds, even when an error occurs mid-scan.
//
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with `fmt.Sprint`.
//
// Pass [UnlimitedMaxNumRows] (or any negative integer) for maxNumRows
// to disable the limit. A value of 0 is enforced as a hard cap that
// permits no data rows: an empty query returns just the header row with
// no error, a non-empty query returns just the header row together with
// [ErrMaxNumRowsExceeded]. Non-negative values cap the number of data
// rows (rows[0] is always the header and is not counted); exceeding the
// cap returns [ErrMaxNumRowsExceeded] along with the header and the
// data rows scanned so far.
//
// On any error (context cancellation, scan failure, or the final rows.Err()),
// the function returns the header plus whatever data rows were scanned
// before the error, together with the wrapped error.
//
// If the query result has no rows, then only the header row
// and no error will be returned.
func QueryRowsAsStrings(ctx context.Context, conn Querier, fmtr QueryFormatter, maxNumRows int, query string, args ...any) (rows [][]string, err error) {
	sqlRows := conn.Query(ctx, query, args...)
	defer func() {
		err = errors.Join(err, sqlRows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, fmtr)
		}
	}()

	if maxNumRows < 0 {
		maxNumRows = math.MaxInt // Practically unlimited
	}

	cols, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}
	rows = [][]string{cols}
	stringScannablePtrs := make([]any, len(cols))
	for sqlRows.Next() {
		if len(rows)-1 >= maxNumRows {
			return rows, ErrMaxNumRowsExceeded{MaxNumRows: maxNumRows}
		}
		if err = ctx.Err(); err != nil {
			return rows, err
		}
		row := make([]string, len(cols))
		// Modify stringScannablePtrs to point to the row values
		for i := range stringScannablePtrs {
			stringScannablePtrs[i] = (*StringScannable)(&row[i])
		}
		if err = sqlRows.Scan(stringScannablePtrs...); err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}
	if err = sqlRows.Err(); err != nil {
		return rows, err
	}
	return rows, nil
}

// QueryStructCallback calls the passed callback with a scanned struct
// for every row returned by the query.
// S must be a struct or pointer to struct type.
// Column values are scanned into struct fields using the provided StructReflector.
//
// If a non nil error is returned from the callback, then this error
// is returned immediately without scanning further rows.
//
// In case of zero rows, no error will be returned.
func QueryStructCallback[S any](ctx context.Context, conn Querier, refl StructReflector, fmtr QueryFormatter, callback func(S) error, query string, args ...any) (err error) {
	defer func() {
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, fmtr)
		}
	}()

	t := reflect.TypeFor[S]()
	st := t
	if st.Kind() == reflect.Pointer {
		st = st.Elem()
	}
	if st.Kind() != reflect.Struct {
		return fmt.Errorf("QueryStructCallback expected struct or pointer to struct type parameter, got %s", t)
	}

	sqlRows := conn.Query(ctx, query, args...)
	defer func() { err = errors.Join(err, sqlRows.Close()) }()

	columns, err := sqlRows.Columns()
	if err != nil {
		return err
	}

	for sqlRows.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var row S
		err = scanStruct(sqlRows, columns, refl, &row)
		if err != nil {
			return err
		}
		err = callback(row)
		if err != nil {
			return err
		}
	}

	return sqlRows.Err()
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
func QueryCallback(ctx context.Context, conn Querier, refl StructReflector, fmtr QueryFormatter, callback any, query string, args ...any) (err error) {
	defer func() {
		if err != nil {
			err = WrapErrorWithQuery(err, query, args, fmtr)
		}
	}()
	val := reflect.ValueOf(callback)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		return fmt.Errorf("QueryCallback expected callback function, got %s", typ)
	}
	if typ.IsVariadic() {
		return fmt.Errorf("QueryCallback callback function must not be variadic: %s", typ)
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
	defer func() { err = errors.Join(err, sqlRows.Close()) }()

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
			err = scanStruct(sqlRows, columns, refl, scannedValPtrs[0])
		} else {
			err = sqlRows.Scan(scannedValPtrs...)
		}
		if err != nil {
			return err
		}

		// Then do callback using reflection
		for i := firstArg; i < typ.NumIn(); i++ {
			callbackArgs[i] = reflect.ValueOf(scannedValPtrs[i-firstArg]).Elem()
		}
		res := val.Call(callbackArgs)
		if len(res) > 0 && !res[0].IsNil() {
			return res[0].Interface().(error)
		}
	}

	return sqlRows.Err()
}
