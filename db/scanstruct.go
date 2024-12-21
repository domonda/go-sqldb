package db

import (
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

// scanStruct scans the srcRow into the destStruct using the reflector.
func scanStruct(srcRow sqldb.Row, reflector StructReflector, destStruct any) error {
	v := reflect.ValueOf(destStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	var (
		setDestStructPtr = false
		destStructPtr    reflect.Value
		newStructPtr     reflect.Value
	)
	if v.Kind() == reflect.Ptr && v.IsNil() && v.CanSet() {
		// Got a nil pointer that we can set with a newly allocated struct
		setDestStructPtr = true
		destStructPtr = v
		newStructPtr = reflect.New(v.Type().Elem())
		// Continue with the newly allocated struct
		v = newStructPtr.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("ScanStruct: expected struct but got %T", destStruct)
	}

	columns, err := srcRow.Columns()
	if err != nil {
		return err
	}

	fieldPointers, err := ReflectStructColumnPointers(v, reflector, columns)
	if err != nil {
		return fmt.Errorf("ScanStruct: %w", err)
	}

	err = srcRow.Scan(fieldPointers...)
	if err != nil {
		return err
	}

	if setDestStructPtr {
		destStructPtr.Set(newStructPtr)
	}

	return nil
}
