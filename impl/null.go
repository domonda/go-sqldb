package impl

import (
	"database/sql/driver"
	"reflect"
)

// IsNull returns if val would be interpreted as NULL by a SQL driver.
// It checks if val is nil, implements driver.Valuer or is a nil pointer, slice, or map.
func IsNull(val interface{}) bool {
	if val == nil {
		return true
	}

	if valuer, ok := val.(driver.Valuer); ok {
		v, _ := valuer.Value()
		return v == nil
	}

	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		return v.IsNil()
	}

	return false
}
