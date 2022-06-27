package sqldb

import (
	"reflect"

	"github.com/domonda/go-sqldb/reflection"
)

type ColumnFilter interface {
	IgnoreColumn(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool
}

type ColumnFilterFunc func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool

func (f ColumnFilterFunc) IgnoreColumn(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return f(name, flags, fieldType, fieldValue)
}

func IgnoreColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, ignore := range names {
			if name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, include := range names {
			if name == include {
				return false
			}
		}
		return true
	})
}

func IgnoreStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, ignore := range names {
			if fieldType.Name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, include := range names {
			if fieldType.Name == include {
				return false
			}
		}
		return true
	})
}

func IgnoreFlags(ignore reflection.StructFieldFlags) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		return flags&ignore != 0
	})
}

var IgnoreHasDefault ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.HasDefault()
})

var IgnorePrimaryKey ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.PrimaryKey()
})

var IgnoreReadOnly ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.ReadOnly()
})

var IgnoreNull ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return IsNull(fieldValue)
})

var IgnoreNullOrZero ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return IsNullOrZero(fieldValue)
})

var IgnoreHasDefaultNullOrZero ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.HasDefault() && IsNullOrZero(fieldValue)
})

type noColumnFilter struct{}

func (noColumnFilter) IgnoreColumn(name string, flags reflection.StructFieldFlags, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return false
}

var AllColumns noColumnFilter
