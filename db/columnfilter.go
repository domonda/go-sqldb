package db

import (
	"reflect"

	"github.com/domonda/go-sqldb"
)

type ColumnFilter interface {
	IgnoreColumn(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool
}

type ColumnFilterFunc func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool

func (f ColumnFilterFunc) IgnoreColumn(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return f(column, fieldType, fieldValue)
}

func IgnoreColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, ignore := range names {
			if column.Name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, include := range names {
			if column.Name == include {
				return false
			}
		}
		return true
	})
}

func IgnoreStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, ignore := range names {
			if fieldType.Name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
		for _, include := range names {
			if fieldType.Name == include {
				return false
			}
		}
		return true
	})
}

var IgnoreHasDefault ColumnFilter = ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return column.HasDefault
})

var IgnorePrimaryKey ColumnFilter = ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return column.PrimaryKey
})

var IgnoreReadOnly ColumnFilter = ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return column.ReadOnly
})

var IgnoreNull ColumnFilter = ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return sqldb.IsNull(fieldValue)
})

var IgnoreNullOrZero ColumnFilter = ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return sqldb.IsNullOrZero(fieldValue)
})

var IgnoreNullOrZeroDefault ColumnFilter = ColumnFilterFunc(func(column Column, fieldType reflect.StructField, fieldValue reflect.Value) bool {
	return column.HasDefault && sqldb.IsNullOrZero(fieldValue)
})
