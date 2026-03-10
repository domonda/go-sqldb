package sqldb

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// TypeMapper is used to map Go types to SQL column types.
type TypeMapper interface {
	// ColumnType returns the SQL column type for the given Go type.
	ColumnType(reflect.Type) string
}

// StagedTypeMapper is a TypeMapper that
// first tries to map a reflect.Type using the Types map,
// if that fails it tries to map the reflect.Kind using the Kinds map,
// and if that fails it calls Default function if it is not nil.
type StagedTypeMapper struct {
	Types   map[reflect.Type]string
	Kinds   map[reflect.Kind]string
	Default func(reflect.Type) string
}

func (m *StagedTypeMapper) ColumnType(t reflect.Type) string {
	if columnType, ok := m.Types[t]; ok {
		return columnType
	}
	if columnType, ok := m.Kinds[t.Kind()]; ok {
		return columnType
	}
	if m.Default != nil {
		return m.Default(t)
	}
	return ""
}

// CreateTableForStruct is mostly used to create tests.
func CreateTableForStruct(ctx context.Context, conn ConnExt, typeMap TypeMapper, rowStruct StructWithTableName) error {
	v := reflect.ValueOf(rowStruct)
	tableName, err := conn.TableNameForStruct(v.Type())
	if err != nil {
		return err
	}
	tableName, err = conn.FormatTableName(tableName)
	if err != nil {
		return err
	}
	columns, fields, err := ReflectStructColumnsAndFields(v, conn)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("CreateTableForStruct %s: no columns at struct %T", tableName, rowStruct)
	}

	var query strings.Builder
	fmt.Fprintf(&query, "CREATE TABLE %s (\n  ", tableName)
	for i := range columns {
		fieldType := fields[i]
		columnName, err := conn.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		columnType := typeMap.ColumnType(fieldType)
		if columnType == "" {
			return fmt.Errorf("CreateTableForStruct %s: no column type for field %s of type %s", tableName, columnName, fieldType)
		}
		if i > 0 {
			query.WriteString(",\n  ")
		}
		fmt.Fprint(&query, columnName, " ", columnType)
		if columns[i].PrimaryKey {
			query.WriteString(" PRIMARY KEY")
		} else if !IsNullable(fieldType) {
			query.WriteString(" NOT NULL")
		}
	}
	query.WriteString("\n)")

	return conn.Exec(ctx, query.String())
}

// CreateTablesAndInsertStructs is mostly used to create tests.
func CreateTablesAndInsertStructs(ctx context.Context, conn ConnExt, typeMap TypeMapper, tables ...[]StructWithTableName) error {
	for _, rows := range tables {
		if len(rows) == 0 {
			continue
		}
		err := CreateTableForStruct(ctx, conn, typeMap, rows[0])
		if err != nil {
			return err
		}
		for _, row := range rows {
			err := InsertRowStruct(ctx, conn, conn, conn, conn, row)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
