package db

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

func ReflectStructValues(structVal reflect.Value, reflector StructReflector, ignoreColumns ...ColumnFilter) (columns []Column, values []any) {
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Type().Field(i)
		column, use := reflector.MapStructField(structField)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column.IsEmbeddedField() {
			// Embedded struct field
			columnsEmbed, valuesEmbed := ReflectStructValues(fieldValue, reflector, ignoreColumns...)
			columns = append(columns, columnsEmbed...)
			values = append(values, valuesEmbed...)
			continue
		}

		if ignoreColumn(ignoreColumns, column, structField, fieldValue) {
			continue
		}

		columns = append(columns, column)
		values = append(values, fieldValue.Interface())
	}
	return columns, values
}

func ReflectStructFieldTypes(structVal reflect.Value, reflctor StructReflector, ignoreColumns ...ColumnFilter) (columns []Column, fields []reflect.Type) {
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Type().Field(i)
		column, use := reflctor.MapStructField(structField)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column.IsEmbeddedField() {
			columnsEmbed, fieldsEmbed := ReflectStructFieldTypes(fieldValue, reflctor, ignoreColumns...)
			columns = append(columns, columnsEmbed...)
			fields = append(fields, fieldsEmbed...)
			continue
		}

		if ignoreColumn(ignoreColumns, column, structField, fieldValue) {
			continue
		}

		columns = append(columns, column)
		fields = append(fields, structField.Type)
	}
	return columns, fields
}

func ReflectStructColumnPointers(structVal reflect.Value, namer StructReflector, columns []string) (pointers []any, err error) {
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

func reflectStructColumnPointers(structVal reflect.Value, namer StructReflector, columns []string, pointers []any) error {
	structType := structVal.Type()
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		column, use := namer.MapStructField(field)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column.IsEmbeddedField() {
			err := reflectStructColumnPointers(fieldValue, namer, columns, pointers)
			if err != nil {
				return err
			}
			continue
		}

		colIndex := slices.Index(columns, column.Name)
		if colIndex == -1 {
			continue
		}

		if pointers[colIndex] != nil {
			return fmt.Errorf("duplicate mapped column %s onto field %s of struct %s", column.Name, field.Name, structType)
		}

		pointer := fieldValue.Addr().Interface()
		// TODO this should be a Connection implementation detail
		// // If field is a slice or array that does not implement sql.Scanner
		// // and it's not a string scannable []byte type underneath
		// // then wrap it with WrapForArray to make it scannable
		// if NeedsArrayWrappingForScanning(fieldValue) {
		// 	pointer = WrapArray(pointer)
		// }
		pointers[colIndex] = pointer
	}
	return nil
}

func ignoreColumn(filters []ColumnFilter, column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	for _, filter := range filters {
		if filter.IgnoreColumn(column, fieldType, fieldValue) {
			return true
		}
	}
	return false
}

func pkColumnsOfStruct(reflector StructReflector, t reflect.Type) (columns []string, err error) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		column, ok := reflector.MapStructField(field)
		if !ok {
			continue
		}

		if column.Name == "" {
			columnsEmbed, err := pkColumnsOfStruct(reflector, field.Type)
			if err != nil {
				return nil, err
			}
			columns = append(columns, columnsEmbed...)
		} else if column.PrimaryKey {
			// if err = conn.ValidateColumnName(column); err != nil {
			// 	return nil, fmt.Errorf("%w in struct field %s.%s", err, t, field.Name)
			// }
			columns = append(columns, column.Name)
		}
	}
	return columns, nil
}
