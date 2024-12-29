package db

import (
	"fmt"
	"reflect"

	"github.com/domonda/go-sqldb"
)

// scanStruct scans the srcRow into the destStruct using the reflector.
func scanStruct(srcRow sqldb.Row, reflector StructReflector, destStruct reflect.Value) error {
	v := destStruct
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("scanStruct got nil pointer for %s", destStruct.Type())
		}
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
		return fmt.Errorf("scanStruct expected struct or pointer to struct but got %s", destStruct.Type())
	}

	columns, err := srcRow.Columns()
	if err != nil {
		return err
	}

	fieldPointers, err := ReflectStructColumnPointers(v, reflector, columns)
	if err != nil {
		return fmt.Errorf("scanStruct error from ReflectStructColumnPointers: %w", err)
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
