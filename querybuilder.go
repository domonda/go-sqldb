package sqldb

import (
	"io"
)

type QueryBuilder interface {
	QueryForRowWithPK(w io.Writer, table string, pkColumns []string, f QueryFormatter) error
	InsertQuery(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) error
	InsertUniqueQuery(w io.Writer, table string, columns []ColumnInfo, onConflict string, f QueryFormatter) error
	UpsertQuery(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) error
	UpdateValuesQuery(w io.Writer, table string, values Values, where string, args []any, f QueryFormatter) (vals []any, err error)
	UpdateColumnsQuery(w io.Writer, table string, columns []ColumnInfo, f QueryFormatter) error
}
