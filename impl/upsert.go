package impl

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on pkColumn, then an update of the existing row is performed.
func UpsertStruct(ctx context.Context, conn sqldb.Connection, table string, rowStruct interface{}, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) error {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return fmt.Errorf("UpsertStruct to table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("UpsertStruct to table %s: expected struct but got %T", table, rowStruct)
	}

	columns, pkCol, vals := structFields(v, namer, ignoreColumns, restrictToColumns, true)
	if len(columns) == 0 {
		return fmt.Errorf("UpsertStruct to table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}

	var query strings.Builder
	writeInsertQuery(&query, table, columns)
	query.WriteString(` ON CONFLICT(`)
	first := true
	for i := range columns {
		if !pkCol[i] {
			continue
		}
		if first {
			first = false
		} else {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, `"%s"`, columns[i])
	}
	if first {
		return fmt.Errorf("UpsertStruct to table %s: %T has no exported struct fields with ,pk tag value suffix to mark primary key column(s)", table, rowStruct)
	}

	query.WriteString(`) DO UPDATE SET `)
	first = true
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

	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return fmt.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}
