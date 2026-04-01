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
func InsertReturning(ctx context.Context, conn Querier, refl StructReflector, builder ReturningQueryBuilder, fmtr QueryFormatter, table string, values Values, returning string) *Row {
	if len(values) == 0 {
		return NewRow(NewErrRows(fmt.Errorf("InsertReturning into table %s: no values", table)), refl, fmtr, "", nil)
	}
	cols, vals := values.SortedColumnsAndValues()
	query, err := builder.InsertReturning(fmtr, table, cols, returning)
	if err != nil {
		return NewRow(NewErrRows(fmt.Errorf("failed to create INSERT RETURNING query: %w", err)), refl, fmtr, "", nil)
	}
	rows := conn.Query(ctx, query, vals...)
	return NewRow(rows, refl, fmtr, query, vals)
}

// InsertUnique inserts a new row into the table using the passed values
// or does nothing if the onConflict statement applies.
// Returns true if a row was inserted.
func InsertUnique(ctx context.Context, conn Executor, builder UpsertQueryBuilder, fmtr QueryFormatter, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	cols, vals := values.SortedColumnsAndValues()
	query, err := builder.InsertUnique(fmtr, table, cols, onConflict)
	if err != nil {
		return false, fmt.Errorf("failed to create INSERT query: %w", err)
	}

	n, err := conn.ExecRowsAffected(ctx, query, vals...)
	if err != nil {
		return false, WrapErrorWithQuery(err, query, vals, fmtr)
	}
	return n > 0, nil
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
	columns, cached.structFieldIndices, vals, err = refl.ReflectStructColumnsFieldIndicesAndValues(structVal, append(options, IgnoreReadOnly)...)
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
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	table, err := refl.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	options = append(options, IgnoreReadOnly)
	columns, err := refl.ReflectStructColumns(structType, options...)
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
		vals, err := refl.ReflectStructValues(strct, options...)
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
func InsertUniqueRowStruct(ctx context.Context, conn Executor, refl StructReflector, builder UpsertQueryBuilder, fmtr QueryFormatter, rowStruct StructWithTableName, onConflict string, options ...QueryOption) (inserted bool, err error) {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return false, err
	}

	table, err := refl.TableNameForStruct(structVal.Type())
	if err != nil {
		return false, err
	}

	columns, vals, err := refl.ReflectStructColumnsAndValues(structVal, append(options, IgnoreReadOnly)...)
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

	n, err := conn.ExecRowsAffected(ctx, query, vals...)
	if err != nil {
		return false, WrapErrorWithQuery(err, query, vals, fmtr)
	}
	return n > 0, nil
}

// InsertRowStructs inserts a slice of structs as new rows into the table for the given struct type.
// Rows are batched into multi-row INSERT statements respecting the driver's MaxArgs() limit
// (e.g. 65,535 for PostgreSQL, 2,100 for SQL Server).
//
// Optimization strategy:
//   - Single row: delegates to [InsertRowStruct] (benefits from the query cache).
//   - Single batch (all rows fit within MaxArgs): executes a single multi-row INSERT directly
//     without a transaction or prepared statement.
//   - Multiple batches: wraps all batches in a transaction for atomicity.
//     When there are 2+ full batches, a prepared statement is created and reused
//     across all full batches to avoid repeated query parsing on the server.
//     Any remainder rows are executed as a separate, smaller multi-row INSERT.
//
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStructs[S StructWithTableName](ctx context.Context, conn Connection, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStructs []S, options ...QueryOption) error {
	numTotalRows := len(rowStructs)
	switch numTotalRows {
	case 0:
		return nil
	case 1:
		return InsertRowStruct(ctx, conn, refl, builder, fmtr, rowStructs[0], options...)
	}

	options = append(options, IgnoreReadOnly)
	structType := reflect.TypeFor[S]()
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	columns, err := refl.ReflectStructColumns(structType, options...)
	if err != nil {
		return err
	}
	table, err := refl.TableNameForStruct(structType)
	if err != nil {
		return err
	}

	numCols := len(columns)
	if numCols == 0 {
		return fmt.Errorf("InsertRowStructs: no columns mapped for struct %s", structType)
	}
	rowsPerBatch := fmtr.MaxArgs() / numCols
	if rowsPerBatch < 1 {
		return fmt.Errorf("InsertRowStructs: MaxArgs() %d is less than number of columns %d", fmtr.MaxArgs(), numCols)
	}
	if rowsPerBatch > numTotalRows {
		rowsPerBatch = numTotalRows
	}

	numFullBatches := numTotalRows / rowsPerBatch
	numRemainderRows := numTotalRows % rowsPerBatch

	collectValues := func(start, end int) ([]any, error) {
		vals := make([]any, 0, (end-start)*numCols)
		for i := start; i < end; i++ {
			structVal, err := derefStruct(reflect.ValueOf(rowStructs[i]))
			if err != nil {
				return nil, err
			}
			rowVals, err := refl.ReflectStructValues(structVal, options...)
			if err != nil {
				return nil, err
			}
			vals = append(vals, rowVals...)
		}
		return vals, nil
	}

	// All rows fit in a single batch: no transaction, no prepare
	if numFullBatches <= 1 && numRemainderRows == 0 {
		query, err := builder.InsertRows(fmtr, table, columns, numTotalRows)
		if err != nil {
			return fmt.Errorf("failed to create INSERT query: %w", err)
		}
		vals, err := collectValues(0, numTotalRows)
		if err != nil {
			return err
		}
		err = conn.Exec(ctx, query, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, fmtr)
		}
		return nil
	}

	// Multiple batches: wrap in a transaction
	return Transaction(ctx, conn, nil, func(tx Connection) error {
		fullBatchQuery, err := builder.InsertRows(fmtr, table, columns, rowsPerBatch)
		if err != nil {
			return fmt.Errorf("failed to create INSERT query: %w", err)
		}

		switch {
		case numFullBatches >= 2:
			// Prepare the full-batch query for repeated execution
			stmt, err := tx.Prepare(ctx, fullBatchQuery)
			if err != nil {
				return fmt.Errorf("failed to prepare INSERT query: %w", err)
			}
			var execErr error
			for batch := range numFullBatches {
				start := batch * rowsPerBatch
				vals, err := collectValues(start, start+rowsPerBatch)
				if err != nil {
					execErr = err
					break
				}
				err = stmt.Exec(ctx, vals...)
				if err != nil {
					execErr = WrapErrorWithQuery(err, fullBatchQuery, vals, fmtr)
					break
				}
			}
			if err := errors.Join(execErr, stmt.Close()); err != nil {
				return err
			}

		case numFullBatches == 1:
			// Single full batch: execute without preparing
			vals, err := collectValues(0, rowsPerBatch)
			if err != nil {
				return err
			}
			err = tx.Exec(ctx, fullBatchQuery, vals...)
			if err != nil {
				return WrapErrorWithQuery(err, fullBatchQuery, vals, fmtr)
			}
		}

		if numRemainderRows == 0 {
			return nil
		}

		remainderQuery, err := builder.InsertRows(fmtr, table, columns, numRemainderRows)
		if err != nil {
			return fmt.Errorf("failed to create INSERT query: %w", err)
		}
		start := numFullBatches * rowsPerBatch
		vals, err := collectValues(start, start+numRemainderRows)
		if err != nil {
			return err
		}
		err = tx.Exec(ctx, remainderQuery, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, remainderQuery, vals, fmtr)
		}
		return nil
	})
}
