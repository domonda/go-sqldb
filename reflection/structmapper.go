package reflection

import (
	"reflect"
)

type StructMapper interface {
	ReflectStructMapping(t reflect.Type) (*StructMapping, error)
}
