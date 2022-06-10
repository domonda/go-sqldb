package db

import (
	"github.com/domonda/go-sqldb"
)

var (
	IgnoreDefault           = sqldb.IgnoreDefault
	IgnorePrimaryKey        = sqldb.IgnorePrimaryKey
	IgnoreReadOnly          = sqldb.IgnoreReadOnly
	IgnoreNull              = sqldb.IgnoreNull
	IgnoreNullOrZero        = sqldb.IgnoreNullOrZero
	IgnoreNullOrZeroDefault = sqldb.IgnoreNullOrZeroDefault
)

func IgnoreColumns(names ...string) sqldb.ColumnFilter {
	return sqldb.IgnoreColumns(names...)
}

func OnlyColumns(names ...string) sqldb.ColumnFilter {
	return sqldb.OnlyColumns(names...)
}

func IgnoreStructFields(names ...string) sqldb.ColumnFilter {
	return sqldb.IgnoreStructFields(names...)
}

func OnlyStructFields(names ...string) sqldb.ColumnFilter {
	return sqldb.OnlyStructFields(names...)
}
