package impl

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/domonda/go-sqldb"
)

func ReflectStructValues(structVal reflect.Value, namer sqldb.StructFieldMapper, ignoreColumns []sqldb.ColumnFilter) (columns []string, pkCols []int, values []any) {
	for i := 0; i < structVal.NumField(); i++ {
		fieldType := structVal.Type().Field(i)
		_, column, flags, use := namer.MapStructField(fieldType)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column == "" {
			// Embedded struct field
			columnsEmbed, pkColsEmbed, valuesEmbed := ReflectStructValues(fieldValue, namer, ignoreColumns)
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

		if flags.PrimaryKey() {
			pkCols = append(pkCols, len(columns))
		}
		columns = append(columns, column)
		values = append(values, fieldValue.Interface())
	}
	return columns, pkCols, values
}

func ReflectStructColumnPointers(structVal reflect.Value, namer sqldb.StructFieldMapper, columns []string) (pointers []any, err error) {
	if len(columns) == 0 {
		return nil, errors.New("no columns")
	}
	pointers = make([]any, len(columns))
	err = reflectStructColumnPointers(structVal, namer, columns, pointers)
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

func reflectStructColumnPointers(structVal reflect.Value, namer sqldb.StructFieldMapper, columns []string, pointers []any) error {
	var (
		structType = structVal.Type()
	)
	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i)
		_, column, _, use := namer.MapStructField(fieldType)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column == "" {
			// Embedded struct field
			err := reflectStructColumnPointers(fieldValue, namer, columns, pointers)
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

func ignoreColumn(filters []sqldb.ColumnFilter, name string, flags sqldb.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	for _, filter := range filters {
		if filter.IgnoreColumn(name, flags, fieldType, fieldValue) {
			return true
		}
	}
	return false
}
