package db

import (
	"context"
	"fmt"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
	"golang.org/x/exp/slices"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, table string, values sqldb.Values, where string, args ...any) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}

	conn := Conn(ctx)
	argFmt := conn.ArgFmt()
	query, vals := buildUpdateQuery(table, values, where, argFmt, args)
	err := conn.Exec(query, vals...)
	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}

// UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
// and returning a single row with the columns specified in returning argument.
func UpdateReturningRow(ctx context.Context, table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("UpdateReturningRow table %s: no values passed", table))
	}

	conn := Conn(ctx)
	query, vals := buildUpdateQuery(table, values, where, conn.ArgFmt(), args)
	query += " RETURNING " + returning
	return conn.QueryRow(query, vals...)
}

// UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
// and returning multiple rows with the columns specified in returning argument.
func UpdateReturningRows(ctx context.Context, table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	if len(values) == 0 {
		return sqldb.RowsScannerWithError(fmt.Errorf("UpdateReturningRows table %s: no values passed", table))
	}

	conn := Conn(ctx)
	query, vals := buildUpdateQuery(table, values, where, conn.ArgFmt(), args)
	query += " RETURNING " + returning
	return conn.QueryRows(query, vals...)
}

func buildUpdateQuery(table string, values sqldb.Values, where, argFmt string, args []any) (string, []any) {
	names, vals := values.Sorted()

	var query strings.Builder
	fmt.Fprintf(&query, `UPDATE %s SET `, table)
	for i := range names {
		if i > 0 {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, `"%s"=%s`, names[i], fmt.Sprintf(argFmt, 1+len(args)+i))
	}
	fmt.Fprintf(&query, ` WHERE %s`, where)

	return query.String(), append(args, vals...)
}

// UpdateStruct updates a row of table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
func UpdateStruct(ctx context.Context, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	v, err := derefStruct(rowStruct)
	if err != nil {
		return err
	}

	conn := Conn(ctx)
	argFmt := conn.ArgFmt()
	mapper := conn.StructFieldMapper()
	table, columns, pkCols, vals, err := ReflectStructValues(v, mapper, append(ignoreColumns, sqldb.IgnoreReadOnly))
	if err != nil {
		return err
	}
	if table == "" {
		return fmt.Errorf("UpdateStruct: %s has no table name", v.Type())
	}
	if len(pkCols) == 0 {
		return fmt.Errorf("UpdateStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var b strings.Builder
	fmt.Fprintf(&b, `UPDATE %s SET `, table)
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
		fmt.Fprintf(&b, `"%s"=$%d`, columns[i], i+1)
	}

	b.WriteString(` WHERE `)
	for i, pkCol := range pkCols {
		if i > 0 {
			b.WriteString(` AND `)
		}
		fmt.Fprintf(&b, `"%s"=$%d`, columns[pkCol], i+1)
	}

	query := b.String()

	err = conn.Exec(query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}
