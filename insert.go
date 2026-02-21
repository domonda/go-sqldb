package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, c *ConnExt, table string, values Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}
	cols, vals := values.SortedColumnsAndValues()
	query, err := c.QueryBuilder.Insert(c.QueryFormatter, table, cols)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}
	err = c.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, c.QueryFormatter)
	}
	return nil
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, c *ConnExt, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	cols, vals := values.SortedColumnsAndValues()
	query, err := c.QueryBuilder.InsertUnique(c.QueryFormatter, table, cols, onConflict)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	rows := c.Query(ctx, query, vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query, vals, c.QueryFormatter)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}

type queryCache struct {
	query              string
	structFieldIndices [][]int
}

var (
	insertRowStructQueryCache    = make(map[reflect.Type]map[StructReflector]map[QueryBuilder]queryCache)
	insertRowStructQueryCacheMtx sync.RWMutex
)

// InsertRowStruct inserts a new row into table.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, c *ConnExt, rowStruct StructWithTableName, options ...QueryOption) error {
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
		cached, ok := insertRowStructQueryCache[structType][c.StructReflector][c.QueryBuilder]
		insertRowStructQueryCacheMtx.RUnlock()
		if ok {
			vals = make([]any, len(cached.structFieldIndices))
			for i, fieldIndex := range cached.structFieldIndices {
				vals[i] = structVal.FieldByIndex(fieldIndex).Interface()
			}
			err = c.Exec(ctx, cached.query, vals...)
			if err != nil {
				return WrapErrorWithQuery(err, cached.query, vals, c.QueryFormatter)
			}
			return nil
		}
	}
	var cached queryCache
	var columns []ColumnInfo
	columns, cached.structFieldIndices, vals = ReflectStructColumnsFieldIndicesAndValues(structVal, c.StructReflector, append(options, IgnoreReadOnly)...)
	table, err := c.StructReflector.TableNameForStruct(structType)
	if err != nil {
		return err
	}
	cached.query, err = c.QueryBuilder.Insert(c.QueryFormatter, table, columns)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}
	if useCache {
		insertRowStructQueryCacheMtx.Lock()
		if _, ok := insertRowStructQueryCache[structType]; !ok {
			insertRowStructQueryCache[structType] = make(map[StructReflector]map[QueryBuilder]queryCache)
		}
		if _, ok := insertRowStructQueryCache[structType][c.StructReflector]; !ok {
			insertRowStructQueryCache[structType][c.StructReflector] = make(map[QueryBuilder]queryCache)
		}
		insertRowStructQueryCache[structType][c.StructReflector][c.QueryBuilder] = cached
		insertRowStructQueryCacheMtx.Unlock()
	}

	err = c.Exec(ctx, cached.query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, cached.query, vals, c.QueryFormatter)
	}
	return nil
}

func InsertRowStructStmt[S StructWithTableName](ctx context.Context, c *ConnExt, options ...QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	structType := reflect.TypeFor[S]()
	table, err := c.StructReflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	options = append(options, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, c.StructReflector, options...)

	query, err := c.QueryBuilder.Insert(c.QueryFormatter, table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	stmt, err := c.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("can't prepare INSERT query because: %w", err)
	}

	insertFunc = func(ctx context.Context, rowStruct S) error {
		strct, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals := ReflectStructValues(strct, c.StructReflector, options...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, c.QueryFormatter)
		}
		return nil
	}
	return insertFunc, stmt.Close, nil
}

// func InsertStructStmt[S StructWithTableName](ctx context.Context, conn Querier, queryBuilder QueryBuilder, query string) (stmtFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
// 	conn := Conn(ctx)
// 	stmt, err := conn.Prepare(ctx, query)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	stmtFunc = func(ctx context.Context, rowStruct S) error {
// 		TODO
// 		if err != nil {
// 			return WrapErrorWithQuery(err, query, args, conn)
// 		}
// 		return nil
// 	}
// 	return stmtFunc, stmt.Close, nil
// }

// InsertUniqueRowStruct inserts a new row with unique primary key.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns true if a row was inserted.
func InsertUniqueRowStruct(ctx context.Context, c *ConnExt, rowStruct StructWithTableName, onConflict string, options ...QueryOption) (inserted bool, err error) {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return false, err
	}

	table, err := c.StructReflector.TableNameForStruct(structVal.Type())
	if err != nil {
		return false, err
	}

	columns, vals := ReflectStructColumnsAndValues(structVal, c.StructReflector, append(options, IgnoreReadOnly)...)

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	query, err := c.QueryBuilder.InsertUnique(c.QueryFormatter, table, columns, onConflict)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	rows := c.Query(ctx, query, vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query, vals, c.QueryFormatter)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}

// InsertRowStructs inserts a slice structs
// as new rows into table using the DefaultStructReflector.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStructs[S StructWithTableName](ctx context.Context, c *ConnExt, rowStructs []S, options ...QueryOption) error {
	// TODO optimized version that combines multiple structs in one query depending or maxArgs
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return InsertRowStruct(ctx, c, rowStructs[0], options...)
	}
	return TransactionExt(ctx, c, nil, func(tx *ConnExt) (err error) {
		insertFunc, closeStmt, stmtErr := InsertRowStructStmt[S](ctx, tx, options...)
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
