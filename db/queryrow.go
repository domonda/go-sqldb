package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/domonda/go-sqldb"
)

// QueryRowStruct uses the passed pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValues and a table name.
func QueryRowStruct[S any](ctx context.Context, pkValues ...any) (row *S, err error) {
	if len(pkValues) == 0 {
		return nil, errors.New("no primaryKeyValues passed")
	}
	t := reflect.TypeOf(row).Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct template type instead of %s", t)
	}
	conn := Conn(ctx)
	table, pkColumns, err := pkColumnsOfStruct(t, conn.StructFieldNamer())
	if err != nil {
		return nil, err
	}
	if len(pkColumns) != len(pkValues) {
		return nil, fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	query := fmt.Sprintf(`SELECT * FROM %s WHERE "%s" = $1`, table, pkColumns[0])
	for i := 1; i < len(pkColumns); i++ {
		query += fmt.Sprintf(` AND "%s" = $%d`, pkColumns[i], i+1)
	}
	err = conn.QueryRow(query, pkValues...).ScanStruct(&row)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func pkColumnsOfStruct(t reflect.Type, mapper sqldb.StructFieldMapper) (table string, columns []string, err error) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldTable, column, flags, ok := mapper.MapStructField(field)
		if !ok {
			continue
		}
		if fieldTable != "" && fieldTable != table {
			if table != "" {
				return "", nil, fmt.Errorf("table name not unique (%s vs %s) in struct %s", table, fieldTable, t)
			}
			table = fieldTable
		}

		if column == "" {
			fieldTable, columnsEmbed, err := pkColumnsOfStruct(field.Type, mapper)
			if err != nil {
				return "", nil, err
			}
			if fieldTable != "" && fieldTable != table {
				if table != "" {
					return "", nil, fmt.Errorf("table name not unique (%s vs %s) in struct %s", table, fieldTable, t)
				}
				table = fieldTable
			}
			columns = append(columns, columnsEmbed...)
		} else if flags.PrimaryKey() {
			columns = append(columns, column)
		}
	}
	return table, columns, nil
}
