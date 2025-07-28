package db

import (
	"fmt"
	"reflect"

	"github.com/domonda/go-sqldb"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanStruct(row rowScanner, columns []string, reflector sqldb.StructReflector, destStruct any) error {
	v := reflect.ValueOf(destStruct)
	if v.Kind() == reflect.Ptr {
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
	if v.Kind() == reflect.Ptr && v.IsNil() && v.CanSet() {
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

	fieldPointers, err := sqldb.ReflectStructColumnPointers(v, reflector, columns)
	if err != nil {
		return fmt.Errorf("scanStruct error from ReflectStructColumnPointers: %w", err)
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
