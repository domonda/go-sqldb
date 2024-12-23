package db

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, table string, rowStruct any, ignoreColumns ...ColumnFilter) error {
	conn := Conn(ctx)

	table, err := conn.FormatTableName(table)
	if err != nil {
		return err
	}

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

	columns, vals := ReflectStructValues(v, DefaultStructReflector, append(ignoreColumns, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col Column) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpsertStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var query strings.Builder
	err = writeInsertQuery(&query, table, columnNames(columns), conn)
	if err != nil {
		return err
	}
	query.WriteString(` ON CONFLICT(`)
	first := true
	for i := range columns {
		if !columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			query.WriteByte(',')
		}
		columnName, err := conn.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		query.WriteString(columnName)
	}

	query.WriteString(`) DO UPDATE SET`)
	first = true
	for i := range columns {
		if columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			query.WriteByte(',')
		}
		columnName, err := conn.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		fmt.Fprintf(&query, ` %s=%s`, columnName, conn.FormatPlaceholder(i))
	}

	err = conn.Exec(ctx, query.String(), vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return nil
}
