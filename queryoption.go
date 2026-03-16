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

// ColumnFilter is a QueryOption that determines whether a column should be ignored.
type ColumnFilter interface {
	QueryOption

	IgnoreColumn(column *ColumnInfo) bool
}

// QueryOptionsIgnoreColumn reports whether any ColumnFilter in opts ignores the given column.
func QueryOptionsIgnoreColumn(column *ColumnInfo, opts []QueryOption) bool {
	for _, opt := range opts {
		if filter, ok := opt.(ColumnFilter); ok && filter.IgnoreColumn(column) {
			return true
		}
	}
	return false
}

// IgnoreColumnFunc is a function type that implements ColumnFilter.
type IgnoreColumnFunc func(column *ColumnInfo) bool

// QueryOption implements the QueryOption interface.
func (f IgnoreColumnFunc) QueryOption() {}

// IgnoreColumn implements the ColumnFilter interface.
func (f IgnoreColumnFunc) IgnoreColumn(column *ColumnInfo) bool {
	return f(column)
}

// IgnoreHasDefault is an IgnoreColumnFunc that ignores columns with a default value.
var IgnoreHasDefault = IgnoreColumnFunc(func(column *ColumnInfo) bool {
	return column.HasDefault
})

// IgnorePrimaryKey is an IgnoreColumnFunc that ignores primary key columns.
var IgnorePrimaryKey = IgnoreColumnFunc(func(column *ColumnInfo) bool {
	return column.PrimaryKey
})

// IgnoreReadOnly is an IgnoreColumnFunc that ignores read-only columns.
var IgnoreReadOnly = IgnoreColumnFunc(func(column *ColumnInfo) bool {
	return column.ReadOnly
})

// OnlyPrimaryKey is an IgnoreColumnFunc that ignores all columns except primary keys.
var OnlyPrimaryKey = IgnoreColumnFunc(func(column *ColumnInfo) bool {
	return !column.PrimaryKey
})

// IgnoreColumns returns an IgnoreColumnFunc that ignores columns with the given names.
func IgnoreColumns(names ...string) IgnoreColumnFunc {
	return func(column *ColumnInfo) bool {
		return slices.Contains(names, column.Name)
	}
}

// OnlyColumns returns an IgnoreColumnFunc that ignores all columns except those with the given names.
func OnlyColumns(names ...string) IgnoreColumnFunc {
	return func(column *ColumnInfo) bool {
		return !slices.Contains(names, column.Name)
	}
}

///////////////////////////////////////////////////////////////////////////////
// StructFieldFilter

// StructFieldFilter is a QueryOption that determines whether a struct field should be ignored.
type StructFieldFilter interface {
	QueryOption

	IgnoreField(field *reflect.StructField) bool
}

// QueryOptionsIgnoreStructField reports whether any StructFieldFilter in opts ignores the given field.
func QueryOptionsIgnoreStructField(field *reflect.StructField, opts []QueryOption) bool {
	for _, opt := range opts {
		if filter, ok := opt.(StructFieldFilter); ok && filter.IgnoreField(field) {
			return true
		}
	}
	return false
}

// StructFieldFilterFunc is a function type that implements StructFieldFilter.
type StructFieldFilterFunc func(field *reflect.StructField) bool

// QueryOption implements the QueryOption interface.
func (f StructFieldFilterFunc) QueryOption() {}

// IgnoreField implements the StructFieldFilter interface.
func (f StructFieldFilterFunc) IgnoreField(field *reflect.StructField) bool {
	return f(field)
}

// IgnoreStructFields returns a StructFieldFilterFunc that ignores fields with the given names.
func IgnoreStructFields(names ...string) StructFieldFilterFunc {
	return func(field *reflect.StructField) bool {
		return slices.Contains(names, field.Name)
	}
}

// OnlyStructFields returns a StructFieldFilterFunc that ignores all fields except those with the given names.
func OnlyStructFields(names ...string) StructFieldFilterFunc {
	return func(field *reflect.StructField) bool {
		return !slices.Contains(names, field.Name)
	}
}
