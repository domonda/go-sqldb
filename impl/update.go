package impl

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"slices"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, conn Execer, table string, values sqldb.Values, where string, args []any, converter driver.ValueConverter, argFmt string) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}

	query, args := buildUpdateQuery(table, values, where, args)
	return Exec(ctx, conn, query, args, converter, argFmt)
}

// UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
// and returning a single row with the columns specified in returning argument.
func UpdateReturningRow(ctx context.Context, conn Queryer, table string, values sqldb.Values, returning, where string, args []any, converter driver.ValueConverter, argFmt string, mapper sqldb.StructFieldMapper) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("UpdateReturningRow table %s: no values passed", table))
	}

	query, vals := buildUpdateQuery(table, values, where, args)
	query += " RETURNING " + returning
	return QueryRow(ctx, conn, query, vals, converter, argFmt, mapper)
}

// UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
// and returning multiple rows with the columns specified in returning argument.
func UpdateReturningRows(ctx context.Context, conn Queryer, table string, values sqldb.Values, returning, where string, args []any, converter driver.ValueConverter, argFmt string, mapper sqldb.StructFieldMapper) sqldb.RowsScanner {
	if len(values) == 0 {
		return sqldb.RowsScannerWithError(fmt.Errorf("UpdateReturningRows table %s: no values passed", table))
	}

	query, vals := buildUpdateQuery(table, values, where, args)
	query += " RETURNING " + returning
	return QueryRows(ctx, conn, query, vals, converter, argFmt, mapper)
}

func buildUpdateQuery(table string, values sqldb.Values, where string, args []any) (string, []any) {
	// args = WrapArgsForArrays(args)
	names, vals := values.Sorted()

	var query strings.Builder
	fmt.Fprintf(&query, `UPDATE %s SET `, table)
	for i := range names {
		if i > 0 {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, `"%s"=$%d`, names[i], 1+len(args)+i)
	}
	fmt.Fprintf(&query, ` WHERE %s`, where)

	return query.String(), append(args, vals...)
}

// UpdateStruct updates a row of table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
func UpdateStruct(ctx context.Context, conn Execer, table string, rowStruct any, mapper sqldb.StructFieldMapper, ignoreColumns []sqldb.ColumnFilter, converter driver.ValueConverter, argFmt string) error {
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

	columns, pkCols, vals := ReflectStructValues(v, mapper, append(ignoreColumns, sqldb.IgnoreReadOnly))
	if len(pkCols) == 0 {
		return fmt.Errorf("UpdateStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var query strings.Builder
	fmt.Fprintf(&query, `UPDATE %s SET `, table)
	first := true
	for i := range columns {
		if slices.Contains(pkCols, i) {
			continue
		}
		if first {
			first = false
		} else {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, `"%s"=$%d`, columns[i], i+1)
	}

	query.WriteString(` WHERE `)
	for i, pkCol := range pkCols {
		if i > 0 {
			query.WriteString(` AND `)
		}
		fmt.Fprintf(&query, `"%s"=$%d`, columns[pkCol], i+1)
	}

	return Exec(ctx, conn, query.String(), vals, converter, argFmt)
}
