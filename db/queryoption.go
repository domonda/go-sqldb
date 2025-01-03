package db

import (
	"reflect"
)

// QueryOption is the base for all query options
// that are implemented as interfaces including this one.
type QueryOption interface {
	QueryOption()
}

///////////////////////////////////////////////////////////////////////////////
// ColumnFilter

type ColumnFilter interface {
	QueryOption

	IgnoreColumn(column *Column) bool
}

func QueryOptionsIgnoreColumn(column *Column, opts []QueryOption) bool {
	for _, opt := range opts {
		if filter, ok := opt.(ColumnFilter); ok && filter.IgnoreColumn(column) {
			return true
		}
	}
	return false
}

type ColumnFilterFunc func(column *Column) bool

func (f ColumnFilterFunc) QueryOption() {}

func (f ColumnFilterFunc) IgnoreColumn(column *Column) bool {
	return f(column)
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

///////////////////////////////////////////////////////////////////////////////
// StructFieldFilter

type StructFieldFilter interface {
	QueryOption

	IgnoreField(field *reflect.StructField) bool
}

func QueryOptionsIgnoreStructField(field *reflect.StructField, opts []QueryOption) bool {
	for _, opt := range opts {
		if filter, ok := opt.(StructFieldFilter); ok && filter.IgnoreField(field) {
			return true
		}
	}
	return false
}

type StructFieldFilterFunc func(field *reflect.StructField) bool

func (f StructFieldFilterFunc) QueryOption() {}

func (f StructFieldFilterFunc) IgnoreField(field *reflect.StructField) bool {
	return f(field)
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
