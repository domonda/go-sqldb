package reflection

import (
	"reflect"
)

type ColumnFilter interface {
	IgnoreColumn(name string, flags FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool
}
