package db

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, table string, values sqldb.Values, where string, args ...any) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}
	conn := Conn(ctx)

	query, vals, err := buildUpdateQuery(table, values, where, args, conn)
	if err != nil {
		return fmt.Errorf("can't create UPDATE query because: %w", err)
	}
	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query, vals, conn)
	}
	return nil
}

// // UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
// // and returning a single row with the columns specified in returning argument.
// func UpdateReturningRow(ctx context.Context, table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
// 	if len(values) == 0 {
// 		return sqldb.RowScannerWithError(fmt.Errorf("UpdateReturningRow table %s: no values passed", table))
// 	}
// 	conn := Conn(ctx)

// 	query, vals := buildUpdateQuery(table, values, where, args, conn)
// 	query += " RETURNING " + returning
// 	return conn.QueryRow(query, vals...)
// }

// // UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
// // and returning multiple rows with the columns specified in returning argument.
// func UpdateReturningRows(ctx context.Context, table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
// 	if len(values) == 0 {
// 		return sqldb.RowsScannerWithError(fmt.Errorf("UpdateReturningRows table %s: no values passed", table))
// 	}
// 	conn := Conn(ctx)

// 	query, vals := buildUpdateQuery(table, values, where, args, conn)
// 	query += " RETURNING " + returning
// 	return conn.QueryRows(query, vals...)
// }

func buildUpdateQuery(table string, values sqldb.Values, where string, args []any, f sqldb.QueryFormatter) (string, []any, error) {
	table, err := f.FormatTableName(table)
	if err != nil {
		return "", nil, err
	}

	columns, vals := values.Sorted()

	var query strings.Builder
	fmt.Fprintf(&query, `UPDATE %s SET`, table)
	for i, column := range columns {
		column, err = f.FormatColumnName(column)
		if err != nil {
			return "", nil, err
		}
		if i > 0 {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, ` "%s"=%s`, column, f.FormatPlaceholder(len(args)+i))
	}
	fmt.Fprintf(&query, ` WHERE %s`, where)

	return query.String(), append(args, vals...), nil
}

// UpdateStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
func UpdateStruct(ctx context.Context, table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return fmt.Errorf("UpdateStruct of table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("UpdateStruct of table %s: expected struct but got %T", table, rowStruct)
	}

	conn := Conn(ctx)

	columns, pkCols, vals := ReflectStructValues(v, DefaultStructReflectror, append(ignoreColumns, sqldb.IgnoreReadOnly))
	if len(pkCols) == 0 {
		return fmt.Errorf("UpdateStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var b strings.Builder
	fmt.Fprintf(&b, `UPDATE %s SET`, table)
	first := true
	for i := range columns {
		if slices.Contains(pkCols, i) {
			continue
		}
		if first {
			first = false
		} else {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, ` "%s"=%s`, columns[i], conn.FormatPlaceholder(i))
	}

	b.WriteString(` WHERE `)
	for i, pkCol := range pkCols {
		if i > 0 {
			b.WriteString(` AND `)
		}
		fmt.Fprintf(&b, `"%s"=%s`, columns[pkCol], conn.FormatPlaceholder(i))
	}

	query := b.String()

	err := conn.Exec(ctx, query, vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query, vals, conn)
	}
	return nil
}
