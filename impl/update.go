package impl

import (
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(conn sqldb.Connection, table string, values sqldb.Values, where, argFmt string, args []interface{}) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}

	query, vals := buildUpdateQuery(table, values, where, args)
	err := conn.Exec(query, vals...)
	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}

// UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
// and returning a single row with the columns specified in returning argument.
func UpdateReturningRow(conn sqldb.Connection, table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("UpdateReturningRow table %s: no values passed", table))
	}

	query, vals := buildUpdateQuery(table, values, where, args)
	query += " RETURNING " + returning
	return conn.QueryRow(query, vals...)
}

// UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
// and returning multiple rows with the columns specified in returning argument.
func UpdateReturningRows(conn sqldb.Connection, table string, values sqldb.Values, returning, where string, args ...interface{}) sqldb.RowsScanner {
	if len(values) == 0 {
		return sqldb.RowsScannerWithError(fmt.Errorf("UpdateReturningRows table %s: no values passed", table))
	}

	query, vals := buildUpdateQuery(table, values, where, args)
	query += " RETURNING " + returning
	return conn.QueryRows(query, vals...)
}

func buildUpdateQuery(table string, values sqldb.Values, where string, args []interface{}) (string, []interface{}) {
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
func UpdateStruct(conn sqldb.Connection, table string, rowStruct interface{}, namer sqldb.StructFieldNamer, argFmt string, ignoreColumns, restrictToColumns []string) error {
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

	columns, flags, vals := structFieldValues(v, namer, ignoreColumns, restrictToColumns, true)
	if len(columns) == 0 {
		return fmt.Errorf("UpdateStruct of table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}

	var b strings.Builder
	fmt.Fprintf(&b, `UPDATE %s SET `, table)
	first := true
	for i := range columns {
		if f := flags[i]; f.IsPrimaryKey() || f.IsReadOnly() {
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
	hasPK := false
	for i := range columns {
		if !flags[i].IsPrimaryKey() {
			continue
		}
		if !hasPK {
			hasPK = true
		} else {
			b.WriteString(` AND `)
		}
		fmt.Fprintf(&b, `"%s"=$%d`, columns[i], i+1)
	}
	if !hasPK {
		return fmt.Errorf("UpdateStruct of table %s: %T has no exported struct fields with ,pk tag value suffix to mark primary key column(s)", table, rowStruct)
	}
	query := b.String()

	err := conn.Exec(query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}
