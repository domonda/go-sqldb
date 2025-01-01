package db

import (
	"reflect"
)

type ColumnFilter interface {
	IgnoreColumn(column Column, field reflect.StructField) bool
}

type ColumnFilterFunc func(column Column, field reflect.StructField) bool

func (f ColumnFilterFunc) IgnoreColumn(column Column, field reflect.StructField) bool {
	return f(column, field)
}

type ColumnFilters []ColumnFilter

func (filters ColumnFilters) IgnoreColumn(column Column, field reflect.StructField) bool {
	for _, filter := range filters {
		if filter.IgnoreColumn(column, field) {
			return true
		}
	}
	return false
}

func IgnoreColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, field reflect.StructField) bool {
		for _, ignore := range names {
			if column.Name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyColumns(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, field reflect.StructField) bool {
		for _, include := range names {
			if column.Name == include {
				return false
			}
		}
		return true
	})
}

func IgnoreStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, field reflect.StructField) bool {
		for _, ignore := range names {
			if field.Name == ignore {
				return true
			}
		}
		return false
	})
}

func OnlyStructFields(names ...string) ColumnFilter {
	return ColumnFilterFunc(func(column Column, field reflect.StructField) bool {
		for _, include := range names {
			if field.Name == include {
				return false
			}
		}
		return true
	})
}

var IgnoreHasDefault ColumnFilter = ColumnFilterFunc(func(column Column, field reflect.StructField) bool {
	return column.HasDefault
})

var IgnorePrimaryKey ColumnFilter = ColumnFilterFunc(func(column Column, field reflect.StructField) bool {
	return column.PrimaryKey
})

var IgnoreReadOnly ColumnFilter = ColumnFilterFunc(func(column Column, field reflect.StructField) bool {
	return column.ReadOnly
})
