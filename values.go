package sqldb

import "sort"

// Values is a map from column names to values
type Values map[string]interface{}

// Sorted returns the names and values from the Values map
// as separated slices sorted by name.
func (v Values) Sorted() (names []string, values []interface{}) {
	names = make([]string, 0, len(v))
	for name := range v {
		names = append(names, name)
	}
	sort.Strings(names)

	values = make([]interface{}, len(v))
	for i, name := range names {
		values[i] = v[name]
	}

	return names, values
}
