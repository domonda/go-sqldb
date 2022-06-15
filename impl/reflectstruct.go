package impl

import (
	"reflect"

	"github.com/domonda/go-sqldb"
)

func ReflectStructValues(structVal reflect.Value, namer sqldb.StructFieldMapper, ignoreColumns []sqldb.ColumnFilter) (columns []string, pkCols []int, values []any) {
	panic("TODO remove")
}

func ReflectStructColumnPointers(structVal reflect.Value, namer sqldb.StructFieldMapper, columns []string) (pointers []any, err error) {
	panic("TODO remove")
}

func reflectStructColumnPointers(structVal reflect.Value, namer sqldb.StructFieldMapper, columns []string, pointers []any) error {
	panic("TODO remove")
}

func ignoreColumn(filters []sqldb.ColumnFilter, name string, flags sqldb.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	panic("TODO remove")
}
