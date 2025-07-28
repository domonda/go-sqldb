package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, conn Executor, queryBuilder QueryBuilder, table string, values Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}
	cols, vals := values.SortedColumnsAndValues()
	query, err := queryBuilder.Insert(table, cols)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}
	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, queryBuilder)
	}
	return nil
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, conn Querier, queryBuilder QueryBuilder, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	cols, vals := values.SortedColumnsAndValues()
	query, err := queryBuilder.InsertUnique(table, cols, onConflict)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	rows := conn.Query(ctx, query, vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query, vals, queryBuilder)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}

// InsertRowStruct inserts a new row into table.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, conn Executor, queryBuilder QueryBuilder, reflector StructReflector, rowStruct StructWithTableName, options ...QueryOption) error {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}

	table, err := reflector.TableNameForStruct(structVal.Type())
	if err != nil {
		return err
	}

	columns, vals := ReflectStructColumnsAndValues(structVal, reflector, append(options, IgnoreReadOnly)...)

	query, err := queryBuilder.Insert(table, columns)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}

	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, queryBuilder)
	}
	return nil
}

func InsertRowStructStmt[S StructWithTableName](ctx context.Context, conn Preparer, queryBuilder QueryBuilder, reflector StructReflector, options ...QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
	structType := reflect.TypeFor[S]()
	table, err := reflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	options = append(options, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, reflector, options...)

	query, err := queryBuilder.Insert(table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("can't prepare INSERT query because: %w", err)
	}

	insertFunc = func(ctx context.Context, rowStruct S) error {
		strct, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals := ReflectStructValues(strct, reflector, options...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, queryBuilder)
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

// InsertUniqueRowStruct inserts a new row with unique private key.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns true if a row was inserted.
func InsertUniqueRowStruct(ctx context.Context, conn Querier, queryBuilder QueryBuilder, reflector StructReflector, rowStruct StructWithTableName, onConflict string, options ...QueryOption) (inserted bool, err error) {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return false, err
	}

	table, err := reflector.TableNameForStruct(structVal.Type())
	if err != nil {
		return false, err
	}

	columns, vals := ReflectStructColumnsAndValues(structVal, reflector, append(options, IgnoreReadOnly)...)

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	query, err := queryBuilder.InsertUnique(table, columns, onConflict)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	rows := conn.Query(ctx, query, vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query, vals, queryBuilder)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}

// InsertRowStructs inserts a slice structs
// as new rows into table using the DefaultStructReflector.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStructs[S StructWithTableName](ctx context.Context, conn Connection, queryBuilder QueryBuilder, reflector StructReflector, rowStructs []S, options ...QueryOption) error {
	// TODO optimized version that combines multiple structs in one query depending or maxArgs
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return InsertRowStruct(ctx, conn, queryBuilder, reflector, rowStructs[0], options...)
	}
	return Transaction(ctx, conn, nil, func(tx Connection) (e error) {
		insert, done, err := InsertRowStructStmt[S](ctx, tx, queryBuilder, reflector, options...)
		if err != nil {
			return err
		}
		defer func() {
			e = errors.Join(e, done())
		}()

		for _, rowStruct := range rowStructs {
			err = insert(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
