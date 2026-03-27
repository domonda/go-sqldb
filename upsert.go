package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// UpsertRowStruct inserts a new row or updates an existing one
// if inserting conflicts on the primary key column(s).
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The struct must have at least one primary key field.
func UpsertRowStruct(ctx context.Context, conn Executor, refl StructReflector, builder UpsertQueryBuilder, fmtr QueryFormatter, rowStruct StructWithTableName, options ...QueryOption) error {
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
		upsertRowStructQueryCacheMtx.RLock()
		cached, ok := upsertRowStructQueryCache[structType][refl][builder][fmtr]
		upsertRowStructQueryCacheMtx.RUnlock()
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
		return fmt.Errorf("UpsertRowStruct of table %s: %s has no mapped primary key field", table, structType)
	}
	cached.query, err = builder.Upsert(fmtr, table, columns)
	if err != nil {
		return fmt.Errorf("UpsertRowStruct of table %s: failed to create UPSERT query: %w", table, err)
	}
	if useCache {
		upsertRowStructQueryCacheMtx.Lock()
		if _, ok := upsertRowStructQueryCache[structType]; !ok {
			upsertRowStructQueryCache[structType] = make(map[StructReflector]map[UpsertQueryBuilder]map[QueryFormatter]queryCache)
		}
		if _, ok := upsertRowStructQueryCache[structType][refl]; !ok {
			upsertRowStructQueryCache[structType][refl] = make(map[UpsertQueryBuilder]map[QueryFormatter]queryCache)
		}
		if _, ok := upsertRowStructQueryCache[structType][refl][builder]; !ok {
			upsertRowStructQueryCache[structType][refl][builder] = make(map[QueryFormatter]queryCache)
		}
		upsertRowStructQueryCache[structType][refl][builder][fmtr] = cached
		upsertRowStructQueryCacheMtx.Unlock()
	}

	err = conn.Exec(ctx, cached.query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, cached.query, vals, fmtr)
	}
	return nil
}

// UpsertRowStructStmt prepares a statement for upserting rows of type S.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns an upsert function to upsert individual rows and a closeStmt
// function that must be called when done to close the prepared statement.
func UpsertRowStructStmt[S StructWithTableName](ctx context.Context, conn Preparer, refl StructReflector, builder UpsertQueryBuilder, fmtr QueryFormatter, options ...QueryOption) (upsert func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
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
		return nil, nil, fmt.Errorf("UpsertRowStructStmt of table %s: %s has no mapped primary key field", table, structType)
	}

	query, err := builder.Upsert(fmtr, table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertRowStructStmt of table %s: failed to create UPSERT query: %w", table, err)
	}

	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertRowStructStmt of table %s: failed to prepare UPSERT statement: %w", table, err)
	}

	upsert = func(ctx context.Context, rowStruct S) error {
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
	return upsert, stmt.Close, nil
}

// UpsertRowStructs upserts a slice of structs within a transaction
// using a prepared statement for efficiency.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Primary key columns are identified by the "primarykey" option
// in their `db` struct tag (e.g., ID int `db:"id,primarykey"`).
func UpsertRowStructs[S StructWithTableName](ctx context.Context, conn Connection, refl StructReflector, builder UpsertQueryBuilder, fmtr QueryFormatter, rowStructs []S, options ...QueryOption) error {
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return UpsertRowStruct(ctx, conn, refl, builder, fmtr, rowStructs[0], options...)
	}
	return Transaction(ctx, conn, nil, func(tx Connection) (err error) {
		upsertFunc, closeStmt, stmtErr := UpsertRowStructStmt[S](ctx, tx, refl, builder, fmtr, options...)
		if stmtErr != nil {
			return stmtErr
		}
		defer func() {
			err = errors.Join(err, closeStmt())
		}()

		for _, rowStruct := range rowStructs {
			err = upsertFunc(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
