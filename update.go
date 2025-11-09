package sqldb

import (
	"context"
	"fmt"
	"reflect"
	"slices"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, c *ConnExt, table string, values Values, where string, args ...any) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}
	query, vals, err := c.QueryBuilder.Update(c.QueryFormatter, table, values, where, args)
	if err != nil {
		return fmt.Errorf("can't create UPDATE query because: %w", err)
	}
	err = c.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, args, c.QueryFormatter)
	}
	return nil
}

// // UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
// // and returning a single row with the columns specified in returning argument.
// func UpdateReturningRow(ctx context.Context, conn Executor, queryBuilder QueryBuilder, table string, values Values, returning, where string, args ...any) RowScanner {
// 	if len(values) == 0 {
// 		return RowScannerWithError(fmt.Errorf("UpdateReturningRow table %s: no values passed", table))
// 	}
// 	conn := Conn(ctx)

// 	query, vals := buildUpdateQuery(table, values, where, args, conn)
// 	query += " RETURNING " + returning
// 	return conn.QueryRow(query, vals...)
// }

// // UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
// // and returning multiple rows with the columns specified in returning argument.
// func UpdateReturningRows(ctx context.Context, conn Executor, queryBuilder QueryBuilder, table string, values Values, returning, where string, args ...any) RowsScanner {
// 	if len(values) == 0 {
// 		return RowsScannerWithError(fmt.Errorf("UpdateReturningRows table %s: no values passed", table))
// 	}
// 	conn := Conn(ctx)

// 	query, vals := buildUpdateQuery(table, values, where, args, conn)
// 	query += " RETURNING " + returning
// 	return conn.QueryRows(query, vals...)
// }

// UpdateStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
func UpdateStruct(ctx context.Context, c *ConnExt, table string, rowStruct any, options ...QueryOption) error {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Pointer && v.IsNil():
		return fmt.Errorf("UpdateStruct of table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("UpdateStruct of table %s: expected struct but got %T", table, rowStruct)
	}

	columns, vals := ReflectStructColumnsAndValues(v, c.StructReflector, append(options, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpdateStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	query, err := c.QueryBuilder.UpdateColumns(c.QueryFormatter, table, columns)
	if err != nil {
		return err
	}
	err = c.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, c.QueryFormatter)
	}
	return nil
}
