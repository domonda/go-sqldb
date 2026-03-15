package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// Update table row(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, conn Executor, builder QueryBuilder, fmtr QueryFormatter, table string, values Values, where string, args ...any) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}
	query, vals, err := builder.Update(fmtr, table, values, where, args)
	if err != nil {
		return fmt.Errorf("failed to create UPDATE query: %w", err)
	}
	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, fmtr)
	}
	return nil
}

// UpdateReturningRow updates a table row with values using the where clause
// with passed in args starting at $1 and returns a Row for scanning
// the columns specified in the returning argument.
func UpdateReturningRow(ctx context.Context, conn Querier, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, table string, values Values, returning, where string, args ...any) *Row {
	if len(values) == 0 {
		return NewRow(NewErrRows(fmt.Errorf("UpdateReturningRow table %s: no values passed", table)), refl, fmtr, "", nil)
	}
	query, vals, err := builder.Update(fmtr, table, values, where, args)
	if err != nil {
		return NewRow(NewErrRows(fmt.Errorf("failed to create UPDATE query: %w", err)), refl, fmtr, "", nil)
	}
	query += " RETURNING " + returning
	rows := conn.Query(ctx, query, vals...)
	return NewRow(rows, refl, fmtr, query, vals)
}

// UpdateReturningRows updates table rows with values using the where clause
// with passed in args starting at $1 and returns Rows for scanning
// the columns specified in the returning argument.
func UpdateReturningRows(ctx context.Context, conn Querier, builder QueryBuilder, fmtr QueryFormatter, table string, values Values, returning, where string, args ...any) Rows {
	if len(values) == 0 {
		return NewErrRows(fmt.Errorf("UpdateReturningRows table %s: no values passed", table))
	}
	query, vals, err := builder.Update(fmtr, table, values, where, args)
	if err != nil {
		return NewErrRows(fmt.Errorf("failed to create UPDATE query: %w", err))
	}
	query += " RETURNING " + returning
	return conn.Query(ctx, query, vals...)
}

// UpdateRowStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Struct fields can be filtered with options like [IgnoreColumns] or [OnlyColumns].
// The struct must have at least one field with a `db` tag value having a ",primarykey" suffix
// to mark primary key column(s).
func UpdateRowStruct(ctx context.Context, conn Executor, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStruct StructWithTableName, options ...QueryOption) error {
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
		updateRowStructQueryCacheMtx.RLock()
		cached, ok := updateRowStructQueryCache[structType][refl][builder][fmtr]
		updateRowStructQueryCacheMtx.RUnlock()
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
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpdateRowStruct of table %s: %s has no mapped primary key field", table, structType)
	}
	cached.query, err = builder.UpdateColumns(fmtr, table, columns)
	if err != nil {
		return err
	}
	if useCache {
		updateRowStructQueryCacheMtx.Lock()
		if _, ok := updateRowStructQueryCache[structType]; !ok {
			updateRowStructQueryCache[structType] = make(map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryCache)
		}
		if _, ok := updateRowStructQueryCache[structType][refl]; !ok {
			updateRowStructQueryCache[structType][refl] = make(map[QueryBuilder]map[QueryFormatter]queryCache)
		}
		if _, ok := updateRowStructQueryCache[structType][refl][builder]; !ok {
			updateRowStructQueryCache[structType][refl][builder] = make(map[QueryFormatter]queryCache)
		}
		updateRowStructQueryCache[structType][refl][builder][fmtr] = cached
		updateRowStructQueryCacheMtx.Unlock()
	}

	err = conn.Exec(ctx, cached.query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, cached.query, vals, fmtr)
	}
	return nil
}

// UpdateRowStructStmt prepares an UPDATE statement for the struct type S
// and returns a function that executes the update for each row struct.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// The struct must have at least one field with a `db` tag value having a ",primarykey" suffix
// to mark primary key column(s).
// The returned closeStmt function must be called to release the prepared statement.
func UpdateRowStructStmt[S StructWithTableName](ctx context.Context, conn Preparer, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, options ...QueryOption) (updateFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	structType := reflect.TypeFor[S]()
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	table, err := refl.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}

	options = append(options, IgnoreReadOnly)
	columns, err := ReflectStructColumns(structType, refl, options...)
	if err != nil {
		return nil, nil, err
	}
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return nil, nil, fmt.Errorf("UpdateRowStructStmt of table %s: %s has no mapped primary key field", table, structType)
	}

	query, err := builder.UpdateColumns(fmtr, table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("UpdateRowStructStmt of table %s: failed to create UPDATE query: %w", table, err)
	}

	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("UpdateRowStructStmt of table %s: failed to prepare UPDATE statement: %w", table, err)
	}

	updateFunc = func(ctx context.Context, rowStruct S) error {
		v, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals, err := ReflectStructValues(v, refl, options...)
		if err != nil {
			return err
		}
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, fmtr)
		}
		return nil
	}
	return updateFunc, stmt.Close, nil
}

// UpdateRowStructs updates a slice of structs within a transaction
// using a prepared statement for efficiency.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// The struct must have at least one field with a `db` tag value having a ",primarykey" suffix
// to mark primary key column(s).
func UpdateRowStructs[S StructWithTableName](ctx context.Context, conn Connection, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStructs []S, options ...QueryOption) error {
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return UpdateRowStruct(ctx, conn, refl, builder, fmtr, rowStructs[0], options...)
	}
	return Transaction(ctx, conn, nil, func(tx Connection) (err error) {
		updateFunc, closeStmt, stmtErr := UpdateRowStructStmt[S](ctx, tx, refl, builder, fmtr, options...)
		if stmtErr != nil {
			return stmtErr
		}
		defer func() {
			err = errors.Join(err, closeStmt())
		}()

		for _, rowStruct := range rowStructs {
			err = updateFunc(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
