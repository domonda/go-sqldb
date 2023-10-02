package sqldb

import (
	"database/sql/driver"
	"errors"
	"sort"

	"golang.org/x/exp/maps"
)

// Values is a map from column names to values
type Values map[string]any

// Sorted returns the names and values from the Values map
// as separated slices sorted by name.
func (v Values) Sorted() (names []string, values []any) {
	names = maps.Keys(v)
	sort.Strings(names)

	values = make([]any, len(v))
	for i, name := range names {
		values[i] = v[name]
	}

	return names, values
}

func convertValuesInPlace(values []any, converter driver.ValueConverter) error {
	if converter == nil {
		return nil
	}
	var err error
	for i, value := range values {
		v, e := converter.ConvertValue(value)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}
		values[i] = v
	}
	return err
}
