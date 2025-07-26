package sqldb

import (
	"io"
)

type QueryBuilder interface {
	QueryFormatter

	QueryRowWithPK(w io.Writer, table string, pkColumns []string) error
	Insert(w io.Writer, table string, columns []ColumnInfo) error
	InsertUnique(w io.Writer, table string, columns []ColumnInfo, onConflict string) error
	Upsert(w io.Writer, table string, columns []ColumnInfo) error
	UpdateValues(w io.Writer, table string, values Values, where string, args []any) (vals []any, err error)
	UpdateColumns(w io.Writer, table string, columns []ColumnInfo) error
}
