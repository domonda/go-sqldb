package sqldb

import (
	"cmp"
	"slices"
)

type ColumnInfo struct {
	Name       string
	PrimaryKey bool
	HasDefault bool
	ReadOnly   bool
}

func (c *ColumnInfo) IsEmbeddedField() bool {
	return c.Name == ""
}

// Values is a map from column names to values
type Values map[string]any

// SortedColumnsAndValues returns the column names and values
// from the Values map as separated slices sorted by column name.
func (v Values) SortedColumnsAndValues() (columns []ColumnInfo, values []any) {
	columns = make([]ColumnInfo, 0, len(v))
	for name := range v {
		columns = append(columns, ColumnInfo{Name: name})
	}
	slices.SortFunc(columns, func(a, b ColumnInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})
	values = make([]any, len(v))
	for i := range columns {
		values[i] = v[columns[i].Name]
	}
	return columns, values
}
