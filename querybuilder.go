package sqldb

import (
	"io"
)

type QueryBuilder interface {
	QueryForRowWithPK(w io.Writer, table string, pkColumns []string, f QueryFormatter) error
	Insert(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) error
	InsertUnique(w io.Writer, table string, columns []ColumnInfo, onConflict string, f QueryFormatter) error
	Upsert(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) error
	UpdateValues(w io.Writer, table string, values Values, where string, args []any, f QueryFormatter) (vals []any, err error)
	UpdateColumns(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) error
}
