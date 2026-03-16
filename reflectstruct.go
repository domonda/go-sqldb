package sqldb

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func PrimaryKeyColumnsOfStruct(reflector StructReflector, t reflect.Type) (columns []string, err error) {
	rs, err := reflectStruct(reflector, t)
	if err != nil {
		return nil, err
	}
	for _, f := range rs.Fields {
		if f.Column.PrimaryKey {
			columns = append(columns, f.Column.Name)
		}
	}
	return columns, nil
}

func ReflectStructColumnsAndValues(structVal reflect.Value, reflector StructReflector, options ...QueryOption) (columns []ColumnInfo, values []any, err error) {
	rs, err := reflectStruct(reflector, structVal.Type())
	if err != nil {
		return nil, nil, err
	}
	for _, f := range rs.Fields {
		col := f.Column
		sf := f.StructField
		if QueryOptionsIgnoreColumn(&col, options) || QueryOptionsIgnoreStructField(&sf, options) {
			continue
		}
		columns = append(columns, f.Column)
		values = append(values, structVal.FieldByIndex(f.FieldIndex).Interface())
	}
	return columns, values, nil
}

func ReflectStructColumnsFieldIndicesAndValues(structVal reflect.Value, reflector StructReflector, options ...QueryOption) (columns []ColumnInfo, indices [][]int, values []any, err error) {
	rs, err := reflectStruct(reflector, structVal.Type())
	if err != nil {
		return nil, nil, nil, err
	}
	for _, f := range rs.Fields {
		col := f.Column
		sf := f.StructField
		if QueryOptionsIgnoreColumn(&col, options) || QueryOptionsIgnoreStructField(&sf, options) {
			continue
		}
		columns = append(columns, f.Column)
		indices = append(indices, f.FieldIndex)
		values = append(values, structVal.FieldByIndex(f.FieldIndex).Interface())
	}
	return columns, indices, values, nil
}

func ReflectStructValues(structVal reflect.Value, reflector StructReflector, options ...QueryOption) (values []any, err error) {
	rs, err := reflectStruct(reflector, structVal.Type())
	if err != nil {
		return nil, err
	}
	for _, f := range rs.Fields {
		col := f.Column
		sf := f.StructField
		if QueryOptionsIgnoreColumn(&col, options) || QueryOptionsIgnoreStructField(&sf, options) {
			continue
		}
		values = append(values, structVal.FieldByIndex(f.FieldIndex).Interface())
	}
	return values, nil
}

func ReflectStructColumns(structType reflect.Type, reflector StructReflector, options ...QueryOption) (columns []ColumnInfo, err error) {
	rs, err := reflectStruct(reflector, structType)
	if err != nil {
		return nil, err
	}
	for _, f := range rs.Fields {
		col := f.Column
		sf := f.StructField
		if QueryOptionsIgnoreColumn(&col, options) || QueryOptionsIgnoreStructField(&sf, options) {
			continue
		}
		columns = append(columns, f.Column)
	}
	return columns, nil
}

func ReflectStructColumnsAndFields(structVal reflect.Value, reflector StructReflector, options ...QueryOption) (columns []ColumnInfo, fields []reflect.Type, err error) {
	rs, err := reflectStruct(reflector, structVal.Type())
	if err != nil {
		return nil, nil, err
	}
	for _, f := range rs.Fields {
		col := f.Column
		sf := f.StructField
		if QueryOptionsIgnoreColumn(&col, options) || QueryOptionsIgnoreStructField(&sf, options) {
			continue
		}
		columns = append(columns, f.Column)
		fields = append(fields, f.StructField.Type)
	}
	return columns, fields, nil
}

func ReflectStructColumnPointers(structVal reflect.Value, namer StructReflector, columns []string) (pointers []any, err error) {
	if len(columns) == 0 {
		return nil, errors.New("no columns")
	}
	rs, err := reflectStruct(namer, structVal.Type())
	if err != nil {
		return nil, err
	}
	pointers = make([]any, len(columns))
	for i, col := range columns {
		idx, ok := rs.ColumnIndex[col]
		if !ok {
			continue
		}
		pointers[i] = structVal.FieldByIndex(rs.Fields[idx].FieldIndex).Addr().Interface()
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
