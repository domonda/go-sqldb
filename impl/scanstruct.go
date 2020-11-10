package impl

import (
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

func ScanStruct(srcRow Row, destStruct interface{}, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) error {
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
		return fmt.Errorf("ScanStruct: %T has %d fields to scan, but database row has %d columns", destStruct, len(fieldPointers), len(cols))
	}

	dest := make([]interface{}, len(cols))
	for i, col := range cols {
		fieldPtr, ok := fieldPointers[col]
		if !ok {
			return fmt.Errorf("ScanStruct: %T has no target struct field for column %s", destStruct, col)
		}
		dest[i] = fieldPtr
	}

	err = srcRow.Scan(dest...)
	if err != nil {
		return err
	}

	if setDestStructPtr {
		destStructPtr.Set(newStructPtr)
	}

	return nil
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

// structFields returns the struct field names using the passed namer ignoring names in ignoreNames
// and if restrictToNames is not empty, then filtering out names not in it.
// If true is passed for keepPK, then ignoreNames and restrictToNames are not applied to names with
// the ,pk suffix in their struct field naming tag.
// The same number of pkCol bools will be returend as names, every corresponding bool marking
// if the name had the ,pk suffix in their struct field naming tag.
func structFields(v reflect.Value, namer sqldb.StructFieldNamer, ignoreNames, restrictToNames []string, keepPK bool) (names []string, pkCol []bool, vals []interface{}) {
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		fieldValue := v.Field(i)
		switch {
		case fieldType.Anonymous:
			embedNames, embedPKs, embedValues := structFields(fieldValue, namer, ignoreNames, restrictToNames, keepPK)
			names = append(names, embedNames...)
			pkCol = append(pkCol, embedPKs...)
			vals = append(vals, embedValues...)

		case fieldType.PkgPath == "":
			name, isPK := namer.StructFieldName(fieldType)
			if validName(name, ignoreNames, restrictToNames) || (isPK && keepPK && validName(name, nil, nil)) {
				names = append(names, name)
				pkCol = append(pkCol, isPK)
				vals = append(vals, fieldValue.Interface())
			}
		}
	}
	return names, pkCol, vals
}

// validName returns if a name not empty or not "-" and not in ignoreNames
// and if restrictToNames is not empty, then not in restrictToNames.
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
