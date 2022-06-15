package reflection

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/exp/slices"
)

func ReflectStructValues(structVal reflect.Value, mapper StructFieldMapper, ignoreColumns []ColumnFilter) (table string, columns []string, pkCols []int, values []any, err error) {
	structType := structVal.Type()
	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i)
		fieldTable, column, flags, use := mapper.MapStructField(fieldType)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column == "" {
			// Embedded struct field
			fieldTable, columnsEmbed, pkColsEmbed, valuesEmbed, err := ReflectStructValues(fieldValue, mapper, ignoreColumns)
			if err != nil {
				return "", nil, nil, nil, err
			}
			if fieldTable != "" && fieldTable != table {
				if table != "" {
					return "", nil, nil, nil, fmt.Errorf("table name not unique (%s vs %s) in struct %s", table, fieldTable, structType)
				}
				table = fieldTable
			}
			for _, pkCol := range pkColsEmbed {
				pkCols = append(pkCols, pkCol+len(columns))
			}
			columns = append(columns, columnsEmbed...)
			values = append(values, valuesEmbed...)
			continue
		}

		if ignoreColumn(ignoreColumns, column, flags, fieldType, fieldValue) {
			continue
		}

		if fieldTable != "" && fieldTable != table {
			if table != "" {
				return "", nil, nil, nil, fmt.Errorf("table name not unique (%s vs %s) in struct %s", table, fieldTable, structType)
			}
			table = fieldTable
		}
		if flags.PrimaryKey() {
			pkCols = append(pkCols, len(columns))
		}
		columns = append(columns, column)
		values = append(values, fieldValue.Interface())
	}
	return table, columns, pkCols, values, nil
}

func ReflectStructColumnPointers(structVal reflect.Value, mapper StructFieldMapper, columns []string) (pointers []any, err error) {
	if len(columns) == 0 {
		return nil, errors.New("no columns")
	}
	pointers = make([]any, len(columns))
	err = reflectStructColumnPointers(structVal, mapper, columns, pointers)
	if err != nil {
		return nil, err
	}
	for _, ptr := range pointers {
		if ptr != nil {
			continue
		}
		nilCols := new(strings.Builder)
		for i, ptr := range pointers {
			if ptr != nil {
				continue
			}
			if nilCols.Len() > 0 {
				nilCols.WriteString(", ")
			}
			fmt.Fprintf(nilCols, "column=%s, index=%d", columns[i], i)
		}
		return nil, fmt.Errorf("columns have no mapped struct fields in %s: %s", structVal.Type(), nilCols)
	}
	return pointers, nil
}

func reflectStructColumnPointers(structVal reflect.Value, mapper StructFieldMapper, columns []string, pointers []any) error {
	var (
		structType = structVal.Type()
	)
	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i)
		_, column, _, use := mapper.MapStructField(fieldType)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column == "" {
			// Embedded struct field
			err := reflectStructColumnPointers(fieldValue, mapper, columns, pointers)
			if err != nil {
				return err
			}
			continue
		}

		colIndex := slices.Index(columns, column)
		if colIndex == -1 {
			continue
		}

		if pointers[colIndex] != nil {
			return fmt.Errorf("duplicate mapped column %s onto field %s of struct %s", column, fieldType.Name, structType)
		}

		pointers[colIndex] = fieldValue.Addr().Interface()
	}
	return nil
}

func ignoreColumn(filters []ColumnFilter, name string, flags FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	for _, filter := range filters {
		if filter.IgnoreColumn(name, flags, fieldType, fieldValue) {
			return true
		}
	}
	return false
}
