package reflection

import (
	"reflect"
)

type ColumnFilter interface {
	IgnoreColumn(*StructColumn, reflect.Value) bool
}
