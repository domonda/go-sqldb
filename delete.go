package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// DeleteRowStruct deletes a row from the table identified by the primary key columns
// of the given struct. The table name is derived from the `db` struct tag of an embedded
// sqldb.TableName field (e.g., sqldb.TableName `db:"my_table"`).
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns a wrapped [sql.ErrNoRows] error if no row was affected by the delete.
func DeleteRowStruct(ctx context.Context, conn Executor, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStruct StructWithTableName) error {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}
	structType := structVal.Type()

	var vals []any
	// Use the cache (no user options to vary the key)
	deleteRowStructQueryCacheMtx.RLock()
	cached, ok := deleteRowStructQueryCache[structType][refl][builder][fmtr]
	deleteRowStructQueryCacheMtx.RUnlock()
	if ok {
		vals = make([]any, len(cached.structFieldIndices))
		for i, fieldIndex := range cached.structFieldIndices {
			vals[i] = structVal.FieldByIndex(fieldIndex).Interface()
		}
		n, err := conn.ExecRowsAffected(ctx, cached.query, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, cached.query, vals, fmtr)
		}
		if n == 0 {
			return WrapErrorWithQuery(sql.ErrNoRows, cached.query, vals, fmtr)
		}
		return nil
	}

	var columns []ColumnInfo
	columns, cached.structFieldIndices, vals, err = refl.ReflectStructColumnsFieldIndicesAndValues(structVal, OnlyPrimaryKey)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		table, _ := refl.TableNameForStruct(structType)
		return fmt.Errorf("DeleteRowStruct of table %s: %s has no mapped primary key field", table, structType)
	}
	table, err := refl.TableNameForStruct(structType)
	if err != nil {
		return err
	}
	cached.query, err = builder.Delete(fmtr, table, columns)
	if err != nil {
		return fmt.Errorf("DeleteRowStruct of table %s: failed to create DELETE query: %w", table, err)
	}

	deleteRowStructQueryCacheMtx.Lock()
	if _, ok := deleteRowStructQueryCache[structType]; !ok {
		deleteRowStructQueryCache[structType] = make(map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryCache)
	}
	if _, ok := deleteRowStructQueryCache[structType][refl]; !ok {
		deleteRowStructQueryCache[structType][refl] = make(map[QueryBuilder]map[QueryFormatter]queryCache)
	}
	if _, ok := deleteRowStructQueryCache[structType][refl][builder]; !ok {
		deleteRowStructQueryCache[structType][refl][builder] = make(map[QueryFormatter]queryCache)
	}
	deleteRowStructQueryCache[structType][refl][builder][fmtr] = cached
	deleteRowStructQueryCacheMtx.Unlock()

	n, err := conn.ExecRowsAffected(ctx, cached.query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, cached.query, vals, fmtr)
	}
	if n == 0 {
		return WrapErrorWithQuery(sql.ErrNoRows, cached.query, vals, fmtr)
	}
	return nil
}

// DeleteRowStructStmt prepares a DELETE statement for the struct type S
// and returns a function that executes the delete for each row struct.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// The returned deleteFunc returns a wrapped [sql.ErrNoRows] error
// if no row was affected by the delete.
// The returned closeStmt function must be called to release the prepared statement.
func DeleteRowStructStmt[S StructWithTableName](ctx context.Context, conn Preparer, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter) (deleteFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	structType := reflect.TypeFor[S]()
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	table, err := refl.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}

	columns, err := refl.ReflectStructColumns(structType, OnlyPrimaryKey)
	if err != nil {
		return nil, nil, err
	}
	if len(columns) == 0 {
		return nil, nil, fmt.Errorf("DeleteRowStructStmt of table %s: %s has no mapped primary key field", table, structType)
	}

	query, err := builder.Delete(fmtr, table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("DeleteRowStructStmt of table %s: failed to create DELETE query: %w", table, err)
	}

	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("DeleteRowStructStmt of table %s: failed to prepare DELETE statement: %w", table, err)
	}

	deleteFunc = func(ctx context.Context, rowStruct S) error {
		v, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals, err := refl.ReflectStructValues(v, OnlyPrimaryKey)
		if err != nil {
			return err
		}
		n, err := stmt.ExecRowsAffected(ctx, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, fmtr)
		}
		if n == 0 {
			return WrapErrorWithQuery(sql.ErrNoRows, query, vals, fmtr)
		}
		return nil
	}
	return deleteFunc, stmt.Close, nil
}

// DeleteRowStructs deletes a slice of structs within a transaction
// using a prepared statement for efficiency.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// Returns a wrapped [sql.ErrNoRows] error if no row was affected
// by the delete of any of the structs.
func DeleteRowStructs[S StructWithTableName](ctx context.Context, conn Connection, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, rowStructs []S) error {
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return DeleteRowStruct(ctx, conn, refl, builder, fmtr, rowStructs[0])
	}
	return Transaction(ctx, conn, nil, func(tx Connection) (err error) {
		deleteFunc, closeStmt, stmtErr := DeleteRowStructStmt[S](ctx, tx, refl, builder, fmtr)
		if stmtErr != nil {
			return stmtErr
		}
		defer func() {
			err = errors.Join(err, closeStmt())
		}()

		for _, rowStruct := range rowStructs {
			err = deleteFunc(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
