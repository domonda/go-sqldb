package implhelper

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

func ScanStruct(rows *sql.Rows, rowStruct interface{}, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) error {
	v := reflect.ValueOf(rowStruct)

	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		// sqlx StructScan does not support pointers to nil pointers
		// so set pointer to newly allocated struct
		if v.Kind() == reflect.Ptr && v.IsNil() {
			n := reflect.New(v.Type().Elem())
			// err := s.row.StructScan(n.Interface())
			// if err != nil {
			// 	return err
			// }
			v.Set(n)
			return nil
		}
	}

	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	// TODO new pointer
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return errors.New("ScanStruct: nil struct pointer")
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("ScanStruct: expected struct but got %T", rowStruct)
	}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	fields := make(map[string]interface{}, len(cols))
	err = setStructFieldPointers(v, namer, ignoreColumns, restrictToColumns, fields)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return fmt.Errorf("ScanStruct: %T has no exported struct fields", rowStruct)
	}
	if len(fields) != len(cols) {
		return fmt.Errorf("ScanStruct: %T ", rowStruct)
	}

	dest := make([]interface{}, len(cols))
	for i, col := range cols {
		fieldPtr, ok := fields[col]
		if !ok {
			return fmt.Errorf("ScanStruct: %T has no target struct field for column %s", rowStruct, col)
		}
		dest[i] = fieldPtr
	}

	return rows.Scan(dest...)
}

func setStructFieldPointers(v reflect.Value, namer sqldb.StructFieldNamer, ignoreNames, restrictToNames []string, out map[string]interface{}) error {
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		fieldValue := v.Field(i)
		switch {
		case fieldType.Anonymous:
			err := setStructFieldPointers(fieldValue, namer, ignoreNames, restrictToNames, out)
			if err != nil {
				return err
			}

		case fieldType.PkgPath == "":
			name := namer.StructFieldName(fieldType)
			if validName(name, ignoreNames, restrictToNames) {
				if _, exists := out[name]; exists {
					return fmt.Errorf("ScanStruct: duplicate struct field name or tag %q in %s", name, v.Type())
				}
				out[name] = fieldValue.Addr().Interface()
			}
		}
	}
	return nil
}
