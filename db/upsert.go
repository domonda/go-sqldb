package db

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// UpsertStruct TODO
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, rowStruct StructWithTableName, ignoreColumns ...ColumnFilter) error {
	v, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}
	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(v.Type())
	if err != nil {
		return err
	}
	conn := Conn(ctx)
	table, err = conn.FormatTableName(table)
	if err != nil {
		return err
	}

	columns, vals := ReflectStructColumnsAndValues(v, reflector, append(ignoreColumns, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col Column) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpsertStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var query strings.Builder
	err = writeInsertQuery(&query, table, columns, conn)
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
