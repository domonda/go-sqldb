package db

import (
	"github.com/domonda/go-sqldb"
)

// QueryOption is the base for all query options
// that are implemented as interfaces including this one.
type QueryOption = sqldb.QueryOption

// ColumnInfo holds metadata about a database column
// as mapped from a Go struct field.
type ColumnInfo = sqldb.ColumnInfo

///////////////////////////////////////////////////////////////////////////////
// ColumnFilter

// ColumnFilter is a QueryOption that determines whether a column should be ignored.
type ColumnFilter = sqldb.ColumnFilter

// QueryOptionsIgnoreColumn reports whether any ColumnFilter in opts ignores the given column.
func QueryOptionsIgnoreColumn(column *ColumnInfo, opts []QueryOption) bool {
	return sqldb.QueryOptionsIgnoreColumn(column, opts)
}

// IgnoreColumnFunc is a function type that implements ColumnFilter.
type IgnoreColumnFunc = sqldb.IgnoreColumnFunc

// IgnoreHasDefault is an IgnoreColumnFunc that ignores columns with a default value.
var IgnoreHasDefault = sqldb.IgnoreHasDefault

// IgnorePrimaryKey is an IgnoreColumnFunc that ignores primary key columns.
var IgnorePrimaryKey = sqldb.IgnorePrimaryKey

// IgnoreReadOnly is an IgnoreColumnFunc that ignores read-only columns.
var IgnoreReadOnly = sqldb.IgnoreReadOnly

// OnlyPrimaryKey is an IgnoreColumnFunc that ignores all columns except primary keys.
var OnlyPrimaryKey = sqldb.OnlyPrimaryKey

// IgnoreColumns returns an IgnoreColumnFunc that ignores columns with the given names.
func IgnoreColumns(names ...string) IgnoreColumnFunc {
	return sqldb.IgnoreColumns(names...)
}

// OnlyColumns returns an IgnoreColumnFunc that ignores all columns except those with the given names.
func OnlyColumns(names ...string) IgnoreColumnFunc {
	return sqldb.OnlyColumns(names...)
}
