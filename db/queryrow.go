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
// for the passed pkValues.
func QueryRowStruct[S any](ctx context.Context, table string, pkValues ...any) (row *S, err error) {
	if len(pkValues) == 0 {
		return nil, errors.New("no primaryKeyValues passed")
	}
	t := reflect.TypeOf(row).Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct template type instead of %s", t)
	}
	conn := Conn(ctx)
	pkColumns := pkColumnsOfStruct(t, conn.StructFieldNamer())
	if len(pkColumns) != len(pkValues) {
		return nil, fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	query := fmt.Sprintf(`SELECT * FROM %s WHERE "%s" = $1`, table, pkValues[0])
	for i := 1; i < len(pkValues); i++ {
		query += fmt.Sprintf(` AND "%s" = $%d`, pkValues[i], i+1)
	}
	err = conn.QueryRow(query, pkValues...).ScanStruct(&row)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func pkColumnsOfStruct(t reflect.Type, namer sqldb.StructFieldNamer) (columns []string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name, flags, ok := namer.StructFieldName(field)
		if !ok {
			continue
		}
		if name == "" {
			columns = append(columns, pkColumnsOfStruct(field.Type, namer)...)
		} else if flags.PrimaryKey() {
			columns = append(columns, name)
		}
	}
	return columns
}
