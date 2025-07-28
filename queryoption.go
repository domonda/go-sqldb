package sqldb

import (
	"reflect"
	"slices"
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

	IgnoreColumn(column *ColumnInfo) bool
}

func QueryOptionsIgnoreColumn(column *ColumnInfo, opts []QueryOption) bool {
	for _, opt := range opts {
		if filter, ok := opt.(ColumnFilter); ok && filter.IgnoreColumn(column) {
			return true
		}
	}
	return false
}

type IgnoreColumnFunc func(column *ColumnInfo) bool

func (f IgnoreColumnFunc) QueryOption() {}

func (f IgnoreColumnFunc) IgnoreColumn(column *ColumnInfo) bool {
	return f(column)
}

var IgnoreHasDefault = IgnoreColumnFunc(func(column *ColumnInfo) bool {
	return column.HasDefault
})

var IgnorePrimaryKey = IgnoreColumnFunc(func(column *ColumnInfo) bool {
	return column.PrimaryKey
})

var IgnoreReadOnly = IgnoreColumnFunc(func(column *ColumnInfo) bool {
	return column.ReadOnly
})

func IgnoreColumns(names ...string) IgnoreColumnFunc {
	return func(column *ColumnInfo) bool {
		return slices.Contains(names, column.Name)
	}
}

func OnlyColumns(names ...string) IgnoreColumnFunc {
	return func(column *ColumnInfo) bool {
		return !slices.Contains(names, column.Name)
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
