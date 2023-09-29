package impl

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/domonda/go-sqldb"
)

func ReflectStructValues(structVal reflect.Value, mapper sqldb.StructFieldMapper, ignoreColumns []sqldb.ColumnFilter) (columns []string, pkCols []int, values []any) {
	for i := 0; i < structVal.NumField(); i++ {
		fieldType := structVal.Type().Field(i)
		_, column, flags, use := mapper.MapStructField(fieldType)
		if !use {
			continue
		}
		fieldValue := structVal.Field(i)

		if column == "" {
			// Embedded struct field
			columnsEmbed, pkColsEmbed, valuesEmbed := ReflectStructValues(fieldValue, mapper, ignoreColumns)
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

// ReflectStructColumnPointers uses the passed mapper
// to find the passed columns as fields of the passed struct
// and returns a pointer to a struct field for every mapped column.
//
// If columns and struct fields could not be mapped 1:1 then
// an ErrColumnsWithoutStructFields or ErrStructFieldHasNoColumn
// error is returned together with the successfully mapped pointers.
func ReflectStructColumnPointers(structVal reflect.Value, mapper sqldb.StructFieldMapper, columns []string) (pointers []any, err error) {
	if structVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("got %s instead of a struct", structVal)
	}
	if !structVal.CanAddr() {
		return nil, errors.New("struct can't be addressed")
	}
	if len(columns) == 0 {
		return nil, errors.New("no columns")
	}
	pointers = make([]any, len(columns))
	err = reflectStructColumnPointers(structVal, mapper, columns, pointers)
	if err != nil {
		return nil, err
	}
	// Check if any column could not be mapped onto the struct,
	// indicated by having a nil struct field pointer.
	var nilCols sqldb.ErrColumnsWithoutStructFields
	for i, ptr := range pointers {
		if ptr == nil {
			nilCols.Columns = append(nilCols.Columns, columns[i])
			nilCols.Struct = structVal
		}
	}
	if len(nilCols.Columns) > 0 {
		pointers = slices.DeleteFunc(pointers, func(e any) bool { return e == nil })
		return pointers, nilCols
	}
	return pointers, nil
}

func reflectStructColumnPointers(structVal reflect.Value, mapper sqldb.StructFieldMapper, columns []string, pointers []any) error {
	structType := structVal.Type()
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		_, column, _, use := mapper.MapStructField(field)
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
			return fmt.Errorf("duplicate mapped column %s onto field %s of struct %s", column, field.Name, structType)
		}

		pointer := fieldValue.Addr().Interface()
		// If field is a slice or array that does not implement sql.Scanner
		// and it's not a string scannable []byte type underneath
		// then wrap it with WrapForArray to make it scannable
		if ShouldWrapForArrayScanning(fieldValue) {
			pointer = WrapForArrayScanning(pointer)
		}
		pointers[colIndex] = pointer
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
