package sqldb

import (
	"reflect"

	"github.com/domonda/go-sqldb/reflection"
)

type ColumnFilter interface {
	IgnoreColumn(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool
}

type ColumnFilterFunc func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool

func (f ColumnFilterFunc) IgnoreColumn(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return f(name, flags, fieldType, fieldValue)
}

func IgnoreColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, ignore := range names {
			if name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, include := range names {
			if name == include {
				return false
			}
		}
		return true
	})
}

func IgnoreStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, ignore := range names {
			if fieldType.Name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, include := range names {
			if fieldType.Name == include {
				return false
			}
		}
		return true
	})
}

func IgnoreFlags(ignore reflection.FieldFlag) ColumnFilter {
	return ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		return flags&ignore != 0
	})
}

var IgnoreDefault ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.Default()
})

var IgnorePrimaryKey ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.PrimaryKey()
})

var IgnoreReadOnly ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.ReadOnly()
})

var IgnoreNull ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return IsNull(fieldValue)
})

var IgnoreNullOrZero ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return IsNullOrZero(fieldValue)
})

var IgnoreNullOrZeroDefault ColumnFilter = ColumnFilterFunc(func(name string, flags reflection.FieldFlag, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return flags.Default() && IsNullOrZero(fieldValue)
})
