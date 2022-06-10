package sqldb

import (
	"database/sql/driver"
	"reflect"
)

// IsNull returns if val would be interpreted as NULL by a SQL driver.
// It checks if val is nil, implements driver.Valuer or is a nil pointer, slice, or map.
func IsNull(val any) bool {
	if val == nil {
		return true
	}

	if valuer, ok := val.(driver.Valuer); ok {
		v, err := valuer.Value()
		return v == nil && err == nil
	}

	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		return v.IsNil()
	}

	return false
}

// IsNullOrZero returns if val would be interpreted as NULL by a SQL driver
// or if it is the types zero value
// or if it implements interface{ IsZero() bool } returning true.
func IsNullOrZero(val any) bool {
	if val == nil {
		return true
	}

	if v, ok := val.(interface{ IsZero() bool }); ok && v.IsZero() {
		return true
	}

	if uuid, ok := val.([16]byte); ok {
		var zero [16]byte
		if uuid == zero {
			return true
		}
	}

	if valuer, ok := val.(driver.Valuer); ok {
		v, err := valuer.Value()
		return v == nil && err == nil
	}

	return reflect.ValueOf(val).IsZero()
}
