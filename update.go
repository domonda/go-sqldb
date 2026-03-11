package sqldb

import (
	"context"
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
// Struct fields can be filtered with options like [IgnoreColumns] or [OnlyColumns].
// The struct must have at least one field with a `db` tag value having a ",primarykey" suffix
// to mark primary key column(s).
func UpdateRowStruct(ctx context.Context, conn Executor, refl StructReflector, builder QueryBuilder, fmtr QueryFormatter, table string, rowStruct any, options ...QueryOption) error {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Pointer && v.IsNil():
		return fmt.Errorf("UpdateRowStruct of table %s: unable to update nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("UpdateRowStruct of table %s: expected struct but got %T", table, rowStruct)
	}

	columns, vals, err := ReflectStructColumnsAndValues(v, refl, append(options, IgnoreReadOnly)...)
	if err != nil {
		return err
	}
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpdateRowStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	query, err := builder.UpdateColumns(fmtr, table, columns)
	if err != nil {
		return err
	}
	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, fmtr)
	}
	return nil
}
