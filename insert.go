package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Insert inserts a new row into the table using the passed values.
func Insert(ctx context.Context, conn Executor, builder QueryBuilder, fmtr QueryFormatter, table string, values Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}
	cols, vals := values.SortedColumnsAndValues()
	query, err := builder.Insert(fmtr, table, cols)
	if err != nil {
		return fmt.Errorf("failed to create INSERT query: %w", err)
	}
	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, fmtr)
	}
	return nil
}

// InsertReturning inserts a new row into the table using values
// and returns a Row for scanning the columns listed in the returning clause.
func InsertReturning(ctx context.Context, conn Querier, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, table string, values Values, returning string) *Row {
	if len(values) == 0 {
		return NewRow(NewErrRows(fmt.Errorf("InsertReturning into table %s: no values", table)), refl, fmtr, "", nil)
	}
	cols, vals := values.SortedColumnsAndValues()
	query, err := builder.Insert(fmtr, table, cols)
	if err != nil {
		return NewRow(NewErrRows(fmt.Errorf("failed to create INSERT query: %w", err)), refl, fmtr, "", nil)
	}
	query += " RETURNING " + returning
	rows := conn.Query(ctx, query, vals...)
	return NewRow(rows, refl, fmtr, query, vals)
}

// InsertUnique inserts a new row into the table using the passed values
// or does nothing if the onConflict statement applies.
// Returns true if a row was inserted.
func InsertUnique(ctx context.Context, conn Querier, builder QueryBuilder, fmtr QueryFormatter, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	cols, vals := values.SortedColumnsAndValues()
	query, err := builder.InsertUnique(fmtr, table, cols, onConflict)
	if err != nil {
		return false, fmt.Errorf("failed to create INSERT query: %w", err)
	}

	rows := conn.Query(ctx, query, vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query, vals, fmtr)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}

// InsertRowStruct inserts a new row into the table for the given struct.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, conn Executor, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStruct StructWithTableName, options ...QueryOption) error {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}
	structType := structVal.Type()

	var vals []any
	// Only use the cache when no caller-provided options are passed
	// because options (like ColumnFilter) change which columns are included
	// and the cache key does not account for them.
	useCache := len(options) == 0
	if useCache {
		insertRowStructQueryCacheMtx.RLock()
		cached, ok := insertRowStructQueryCache[structType][refl][builder][fmtr]
		insertRowStructQueryCacheMtx.RUnlock()
		if ok {
			vals = make([]any, len(cached.structFieldIndices))
			for i, fieldIndex := range cached.structFieldIndices {
				vals[i] = structVal.FieldByIndex(fieldIndex).Interface()
			}
			err = conn.Exec(ctx, cached.query, vals...)
			if err != nil {
				return WrapErrorWithQuery(err, cached.query, vals, fmtr)
			}
			return nil
		}
	}
	var cached queryCache
	var columns []ColumnInfo
	columns, cached.structFieldIndices, vals, err = ReflectStructColumnsFieldIndicesAndValues(structVal, refl, append(options, IgnoreReadOnly)...)
	if err != nil {
		return err
	}
	table, err := refl.TableNameForStruct(structType)
	if err != nil {
		return err
	}
	cached.query, err = builder.Insert(fmtr, table, columns)
	if err != nil {
		return fmt.Errorf("failed to create INSERT query: %w", err)
	}
	if useCache {
		insertRowStructQueryCacheMtx.Lock()
		if _, ok := insertRowStructQueryCache[structType]; !ok {
			insertRowStructQueryCache[structType] = make(map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryCache)
		}
		if _, ok := insertRowStructQueryCache[structType][refl]; !ok {
			insertRowStructQueryCache[structType][refl] = make(map[QueryBuilder]map[QueryFormatter]queryCache)
		}
		if _, ok := insertRowStructQueryCache[structType][refl][builder]; !ok {
			insertRowStructQueryCache[structType][refl][builder] = make(map[QueryFormatter]queryCache)
		}
		insertRowStructQueryCache[structType][refl][builder][fmtr] = cached
		insertRowStructQueryCacheMtx.Unlock()
	}

	err = conn.Exec(ctx, cached.query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, cached.query, vals, fmtr)
	}
	return nil
}

// InsertRowStructStmt prepares an INSERT statement for the struct type S
// and returns a function that executes the insert for each row struct.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// The returned closeStmt function must be called to release the prepared statement.
func InsertRowStructStmt[S StructWithTableName](ctx context.Context, conn Preparer, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, options ...QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	structType := reflect.TypeFor[S]()
	table, err := refl.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	options = append(options, IgnoreReadOnly)
	columns, err := ReflectStructColumns(structType, refl, options...)
	if err != nil {
		return nil, nil, err
	}

	query, err := builder.Insert(fmtr, table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create INSERT query: %w", err)
	}

	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare INSERT query: %w", err)
	}

	insertFunc = func(ctx context.Context, rowStruct S) error {
		strct, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals, err := ReflectStructValues(strct, refl, options...)
		if err != nil {
			return err
		}
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, fmtr)
		}
		return nil
	}
	return insertFunc, stmt.Close, nil
}

// InsertUniqueRowStruct inserts a new row or does nothing if the onConflict statement applies.
// Returns true if a row was inserted.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertUniqueRowStruct(ctx context.Context, conn Querier, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStruct StructWithTableName, onConflict string, options ...QueryOption) (inserted bool, err error) {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return false, err
	}

	table, err := refl.TableNameForStruct(structVal.Type())
	if err != nil {
		return false, err
	}

	columns, vals, err := ReflectStructColumnsAndValues(structVal, refl, append(options, IgnoreReadOnly)...)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	query, err := builder.InsertUnique(fmtr, table, columns, onConflict)
	if err != nil {
		return false, fmt.Errorf("failed to create INSERT query: %w", err)
	}

	rows := conn.Query(ctx, query, vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query, vals, fmtr)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}

// InsertRowStructs inserts a slice of structs as new rows into the table for the given struct type.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStructs[S StructWithTableName](ctx context.Context, conn Connection, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStructs []S, options ...QueryOption) error {
	// TODO optimized version that combines multiple structs in one query depending or maxArgs
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return InsertRowStruct(ctx, conn, refl, builder, fmtr, rowStructs[0], options...)
	}
	return Transaction(ctx, conn, nil, func(tx Connection) (err error) {
		insertFunc, closeStmt, stmtErr := InsertRowStructStmt[S](ctx, tx, refl, builder, fmtr, options...)
		if stmtErr != nil {
			return stmtErr
		}
		defer func() {
			err = errors.Join(err, closeStmt())
		}()

		for _, rowStruct := range rowStructs {
			err = insertFunc(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
