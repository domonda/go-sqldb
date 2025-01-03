package db

import (
	"reflect"
)

type ColumnFilter interface {
	IgnoreColumn(column *Column) bool
}

type ColumnFilterFunc func(column *Column) bool

func (f ColumnFilterFunc) IgnoreColumn(column *Column) bool {
	return f(column)
}

type StructFieldFilter interface {
	IgnoreField(field *reflect.StructField) bool
}

type StructFieldFilterFunc func(field *reflect.StructField) bool

func (f StructFieldFilterFunc) IgnoreField(field *reflect.StructField) bool {
	return f(field)
}

type ColumnFilters []ColumnFilter

func (filters ColumnFilters) IgnoreColumn(column *Column) bool {
	for _, filter := range filters {
		if filter.IgnoreColumn(column) {
			return true
		}
	}
	return false
}

func IgnoreColumns(names ...string) ColumnFilterFunc {
	return func(column *Column) bool {
		for _, ignore := range names {
			if column.Name == ignore {
				return true
			}
		}
		return false
	}
}

func OnlyColumns(names ...string) ColumnFilterFunc {
	return func(column *Column) bool {
		for _, include := range names {
			if column.Name == include {
				return false
			}
		}
		return true
	}
}

func IgnoreStructFields(names ...string) StructFieldFilterFunc {
	return func(field *reflect.StructField) bool {
		for _, ignore := range names {
			if field.Name == ignore {
				return true
			}
		}
		return false
	}
}

func OnlyStructFields(names ...string) StructFieldFilterFunc {
	return func(field *reflect.StructField) bool {
		for _, include := range names {
			if field.Name == include {
				return false
			}
		}
		return true
	}
}

var IgnoreHasDefault = ColumnFilterFunc(func(column *Column) bool {
	return column.HasDefault
})

var IgnorePrimaryKey = ColumnFilterFunc(func(column *Column) bool {
	return column.PrimaryKey
})

var IgnoreReadOnly = ColumnFilterFunc(func(column *Column) bool {
	return column.ReadOnly
})
