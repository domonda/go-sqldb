package impl

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-wraperr"
)

// Update a row in table using the passed values and the where statement
// with args starting at $1.
func Update(ctx context.Context, conn sqldb.Connection, table string, values sqldb.Values, where string, args []interface{}) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}

	// where = strings.TrimSpace(where)
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

	err := conn.ExecContext(ctx, query.String(), append(args, vals...)...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}

// UpdateStruct updates a row of table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
func UpdateStruct(ctx context.Context, conn sqldb.Connection, table string, rowStruct interface{}, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) error {
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

	columns, pkCol, vals := structFields(v, namer, ignoreColumns, restrictToColumns, true)
	if len(columns) == 0 {
		return fmt.Errorf("UpdateStruct of table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}

	var query strings.Builder
	fmt.Fprintf(&query, `UPDATE %s SET `, table)
	first := true
	for i := range columns {
		if pkCol[i] {
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
	first = true
	for i := range columns {
		if !pkCol[i] {
			continue
		}
		if first {
			first = false
		} else {
			query.WriteString(` AND `)
		}
		fmt.Fprintf(&query, `"%s"=$%d`, columns[i], i+1)
	}
	if first {
		return fmt.Errorf("UpdateStruct of table %s: %T has no exported struct fields with ,pk tag value suffix to mark primary key column(s)", table, rowStruct)
	}

	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}
