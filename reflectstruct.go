package sqldb

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

func PrimaryKeyColumnsOfStruct(reflector StructReflector, t reflect.Type) (columns []string, err error) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		column, ok := reflector.MapStructField(field)
		if !ok {
			continue
		}

		if column.Name == "" {
			columnsEmbed, err := PrimaryKeyColumnsOfStruct(reflector, field.Type)
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

func ReflectStructColumnsAndValues(structVal reflect.Value, reflector StructReflector, options ...QueryOption) (columns []ColumnInfo, values []any) {
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Type().Field(i)
		column, use := reflector.MapStructField(structField)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column.IsEmbeddedField() {
			// Embedded struct field
			columnsEmbed, valuesEmbed := ReflectStructColumnsAndValues(fieldValue, reflector, options...)
			columns = append(columns, columnsEmbed...)
			values = append(values, valuesEmbed...)
			continue
		}

		if QueryOptionsIgnoreColumn(&column, options) || QueryOptionsIgnoreStructField(&structField, options) {
			continue
		}

		columns = append(columns, column)
		values = append(values, fieldValue.Interface())
	}
	return columns, values
}

func ReflectStructColumnsFieldIndicesAndValues(structVal reflect.Value, reflector StructReflector, options ...QueryOption) (columns []ColumnInfo, indices [][]int, values []any) {
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Type().Field(i)
		column, use := reflector.MapStructField(structField)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column.IsEmbeddedField() {
			// Embedded struct field
			columnsEmbed, indicesEmbed, valuesEmbed := ReflectStructColumnsFieldIndicesAndValues(fieldValue, reflector, options...)
			columns = append(columns, columnsEmbed...)
			for _, embeddedIndex := range indicesEmbed {
				// Prepending the current struct field's index to each embedded index
				indices = append(indices, append(structField.Index, embeddedIndex...))
			}
			values = append(values, valuesEmbed...)
			continue
		}

		if QueryOptionsIgnoreColumn(&column, options) || QueryOptionsIgnoreStructField(&structField, options) {
			continue
		}

		columns = append(columns, column)
		indices = append(indices, structField.Index)
		values = append(values, fieldValue.Interface())
	}
	return columns, indices, values
}

func ReflectStructValues(structVal reflect.Value, reflector StructReflector, options ...QueryOption) (values []any) {
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Type().Field(i)
		column, use := reflector.MapStructField(structField)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column.IsEmbeddedField() {
			// Embedded struct field
			valuesEmbed := ReflectStructValues(fieldValue, reflector, options...)
			values = append(values, valuesEmbed...)
			continue
		}

		if QueryOptionsIgnoreColumn(&column, options) || QueryOptionsIgnoreStructField(&structField, options) {
			continue
		}

		values = append(values, fieldValue.Interface())
	}
	return values
}

func ReflectStructColumns(structType reflect.Type, reflctor StructReflector, options ...QueryOption) (columns []ColumnInfo) {
	for i := 0; i < structType.NumField(); i++ {
		structField := structType.Field(i)
		column, use := reflctor.MapStructField(structField)
		if !use {
			continue
		}

		if column.IsEmbeddedField() {
			columnsEmbed := ReflectStructColumns(structField.Type, reflctor, options...)
			columns = append(columns, columnsEmbed...)
			continue
		}

		if QueryOptionsIgnoreColumn(&column, options) || QueryOptionsIgnoreStructField(&structField, options) {
			continue
		}

		columns = append(columns, column)
	}
	return columns
}

func ReflectStructColumnsAndFields(structVal reflect.Value, reflctor StructReflector, options ...QueryOption) (columns []ColumnInfo, fields []reflect.Type) {
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Type().Field(i)
		column, use := reflctor.MapStructField(structField)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column.IsEmbeddedField() {
			columnsEmbed, fieldsEmbed := ReflectStructColumnsAndFields(fieldValue, reflctor, options...)
			columns = append(columns, columnsEmbed...)
			fields = append(fields, fieldsEmbed...)
			continue
		}

		if QueryOptionsIgnoreColumn(&column, options) || QueryOptionsIgnoreStructField(&structField, options) {
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

func derefStruct(v reflect.Value) (reflect.Value, error) {
	strct := v
	for strct.Kind() == reflect.Pointer {
		if strct.IsNil() {
			return reflect.Value{}, fmt.Errorf("nil pointer %s", v.Type())
		}
		strct = strct.Elem()
	}
	if strct.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected struct or pointer to struct, but got %s", v.Type())
	}
	return strct, nil
}

// isNonSQLScannerStruct returns true if the passed type is a struct
// that does not implement the sql.Scanner interface,
// or a pointer to a struct that does not implement the sql.Scanner interface.
func isNonSQLScannerStruct(t reflect.Type) bool {
	if t == typeOfTime || t.Kind() == reflect.Pointer && t.Elem() == typeOfTime {
		return false
	}
	// Struct that does not implement sql.Scanner
	if t.Kind() == reflect.Struct && !reflect.PointerTo(t).Implements(typeOfSQLScanner) {
		return true
	}
	// Pointer to struct that does not implement sql.Scanner
	if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct && !t.Implements(typeOfSQLScanner) {
		return true
	}
	return false
}
