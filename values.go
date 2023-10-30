package sqldb

import (
	"database/sql/driver"
	"errors"
	"reflect"
	"sort"

	"golang.org/x/exp/maps"
)

// Values implements RowValues
var _ RowValues = Values(nil)

// Values is a map from column names to values.
// It implements RowValues.
type Values map[string]any

// Sorted returns the columns and values from the Values map
// as separated slices sorted by column.
func (v Values) Sorted() (colums []string, values []any) {
	colums = v.Columns()
	values = make([]any, len(v))
	for i, col := range colums {
		values[i] = v[col]
	}
	return colums, values
}

func (v Values) Columns() []string {
	cols := maps.Keys(v)
	sort.Strings(cols)
	return cols
}

func (v Values) RowValues() ([]any, error) {
	values := make([]any, len(v))
	for i, col := range v.Columns() {
		values[i] = v[col]
	}
	return values, nil
}

func mapKeysAndValues(v reflect.Value) (keys []string, values []any) {
	k := v.MapKeys()
	sort.Slice(k, func(i, j int) bool {
		return k[i].String() < k[j].String()
	})

	keys = make([]string, len(k))
	for i, key := range k {
		keys[i] = key.String()
	}

	values = make([]any, len(k))
	for i, key := range k {
		values[i] = v.MapIndex(key).Interface()
	}

	return keys, values
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
