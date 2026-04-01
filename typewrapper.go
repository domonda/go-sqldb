package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
)

// TypeWrapper wraps Go values as sql.Scanner or driver.Valuer
// to customize how types are read from and written to the database.
// Implementations return nil if they don't handle the given value's type,
// allowing multiple TypeWrappers to be composed via TypeWrappers.
type TypeWrapper interface {
	// WrapAsScanner returns a sql.Scanner that wraps the given value.
	// Returns nil if the value cannot be wrapped as a scanner.
	WrapAsScanner(val reflect.Value) sql.Scanner

	// WrapAsValuer returns a driver.Valuer that wraps the given value.
	// Returns nil if the value cannot be wrapped as a valuer.
	WrapAsValuer(val reflect.Value) driver.Valuer
}

// Ensure that TypeWrappers implements TypeWrapper
var _ TypeWrapper = (*TypeWrappers)(nil)

// TypeWrappers is a slice of TypeWrapper that itself implements TypeWrapper.
// It iterates through its elements and returns the result of the first
// TypeWrapper that returns a non-nil value.
type TypeWrappers []TypeWrapper

func (tws TypeWrappers) WrapAsScanner(val reflect.Value) sql.Scanner {
	for _, tw := range tws {
		if w := tw.WrapAsScanner(val); w != nil {
			return w
		}
	}
	return nil
}

func (tws TypeWrappers) WrapAsValuer(val reflect.Value) driver.Valuer {
	for _, tw := range tws {
		if w := tw.WrapAsValuer(val); w != nil {
			return w
		}
	}
	return nil
}
