package impl

import (
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

func ScanStruct(srcRow Row, destStruct interface{}, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) (err error) {
	v := reflect.ValueOf(destStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	if v.Kind() == reflect.Ptr && v.IsNil() && v.CanSet() {
		// Got a pointer to a pointer that we can set with a newly allocated struct
		structPtr := reflect.New(v.Type().Elem())
		defer func() {
			// but set only after scanning into the struct
			// to leave it unchanged in case of an error
			if err == nil {
				v.Set(structPtr)
			}
		}()

		v = structPtr.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("ScanStruct: expected struct but got %T", destStruct)
	}

	cols, err := srcRow.Columns()
	if err != nil {
		return err
	}

	fieldPointers := make(map[string]interface{}, len(cols))
	err = getStructFieldPointers(v, namer, ignoreColumns, restrictToColumns, fieldPointers)
	if err != nil {
		return err
	}
	if len(fieldPointers) == 0 {
		return fmt.Errorf("ScanStruct: %T has no exported struct fieldPointers", destStruct)
	}
	if len(fieldPointers) != len(cols) {
		return fmt.Errorf("ScanStruct: %T ", destStruct)
	}

	dest := make([]interface{}, len(cols))
	for i, col := range cols {
		fieldPtr, ok := fieldPointers[col]
		if !ok {
			return fmt.Errorf("ScanStruct: %T has no target struct field for column %s", destStruct, col)
		}
		dest[i] = fieldPtr
	}

	return srcRow.Scan(dest...)
}

func getStructFieldPointers(v reflect.Value, namer sqldb.StructFieldNamer, ignoreNames, restrictToNames []string, outFieldPtrs map[string]interface{}) error {
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		fieldValue := v.Field(i)
		switch {
		case fieldType.Anonymous:
			err := getStructFieldPointers(fieldValue, namer, ignoreNames, restrictToNames, outFieldPtrs)
			if err != nil {
				return err
			}

		case fieldType.PkgPath == "":
			name, _ := namer.StructFieldName(fieldType)
			if validName(name, ignoreNames, restrictToNames) {
				if _, exists := outFieldPtrs[name]; exists {
					return fmt.Errorf("ScanStruct: duplicate struct field name or tag %q in %s", name, v.Type())
				}
				outFieldPtrs[name] = fieldValue.Addr().Interface()
			}
		}
	}
	return nil
}

func validName(name string, ignoreNames, restrictToNames []string) bool {
	if name == "" || name == "-" {
		return false
	}
	for _, ignore := range ignoreNames {
		if name == ignore {
			return false
		}
	}
	if len(restrictToNames) == 0 {
		return true
	}
	for _, allowed := range restrictToNames {
		if name == allowed {
			return true
		}
	}
	return false
}
