package sqldb

import (
	"fmt"
	"reflect"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanStruct(row rowScanner, columns []string, reflector StructReflector, destStruct any) error {
	if reflector == nil {
		return fmt.Errorf("scanStruct got nil StructReflector")
	}
	v := reflect.ValueOf(destStruct)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return fmt.Errorf("scanStruct got nil pointer for %T", destStruct)
		}
		v = v.Elem()
	}

	var (
		setDestStructPtr = false
		destStructPtr    reflect.Value
		newStructPtr     reflect.Value
	)
	if v.Kind() == reflect.Pointer && v.IsNil() && v.CanSet() {
		// Got a nil pointer that we can set with a newly allocated struct
		setDestStructPtr = true
		destStructPtr = v
		newStructPtr = reflect.New(v.Type().Elem())
		// Continue with the newly allocated struct
		v = newStructPtr.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("scanStruct expected struct or pointer to struct but got %T", destStruct)
	}

	fieldPointers, err := reflector.ScanableStructFieldsForColumns(v, columns)
	if err != nil {
		return fmt.Errorf("scanStruct error from ScanableStructFieldsForColumns: %w", err)
	}
	err = row.Scan(fieldPointers...)
	if err != nil {
		return err
	}
	if setDestStructPtr {
		destStructPtr.Set(newStructPtr)
	}
	return nil
}

// isNonSQLScannerStruct returns true if the passed type is a struct
// that does not implement the sql.Scanner interface,
// or a pointer to a struct that does not implement the sql.Scanner interface.
func isNonSQLScannerStruct(t reflect.Type) bool {
	if t == typeOfTime || t.Kind() == reflect.Pointer && t.Elem() == typeOfTime {
		return false
	}
	// Struct that does not implement sql.Scanner
	if t.Kind() == reflect.Struct && !reflect.PointerTo(t).Implements(typeOfSQLScanner) {
		return true
	}
	// Pointer to struct that does not implement sql.Scanner
	if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct && !t.Implements(typeOfSQLScanner) {
		return true
	}
	return false
}

func derefStruct(v reflect.Value) (reflect.Value, error) {
	strct := v
	for strct.Kind() == reflect.Pointer {
		if strct.IsNil() {
			return reflect.Value{}, fmt.Errorf("nil pointer %s", v.Type())
		}
		strct = strct.Elem()
	}
	if strct.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected struct or pointer to struct, but got %s", v.Type())
	}
	return strct, nil
}
