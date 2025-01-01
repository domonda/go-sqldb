package db

import (
	"cmp"
	"slices"
)

// Values is a map from column names to values
type Values map[string]any

// SortedColumnsAndValues returns the names and values from the Values map
// as separated slices sorted by name.
func (v Values) SortedColumnsAndValues() (columns []Column, values []any) {
	columns = make([]Column, 0, len(v))
	for name := range v {
		columns = append(columns, Column{Name: name})
	}
	slices.SortFunc(columns, func(a, b Column) int {
		return cmp.Compare(a.Name, b.Name)
	})
	values = make([]any, len(v))
	for i := range columns {
		values[i] = v[columns[i].Name]
	}
	return columns, values
}
