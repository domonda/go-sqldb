package impl

import (
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

func ScanStruct(srcRow Row, destStruct any, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) error {
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

	fieldPointers := make(map[string]any, len(cols))
	err = getStructFieldPointers(v, namer, ignoreColumns, restrictToColumns, fieldPointers)
	if err != nil {
		return err
	}
	if len(fieldPointers) == 0 {
		return fmt.Errorf("ScanStruct: %T has no exported struct fieldPointers", destStruct)
	}
	if len(fieldPointers) != len(cols) {
		return fmt.Errorf("ScanStruct: %T has %d fields to scan, but database row has %d columns: %v", destStruct, len(fieldPointers), len(cols), cols)
	}

	dest := make([]any, len(cols))
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

func getStructFieldPointers(v reflect.Value, namer sqldb.StructFieldNamer, ignoreNames, restrictToNames []string, outFieldPtrs map[string]any) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		name, _, ok := namer.StructFieldName(field)
		if !ok {
			continue
		}

		if field.Anonymous {
			err := getStructFieldPointers(v.Field(i), namer, ignoreNames, restrictToNames, outFieldPtrs)
			if err != nil {
				return err
			}
			continue
		}

		if !validName(name, ignoreNames, restrictToNames) {
			continue
		}
		if outFieldPtrs[name] != nil {
			return fmt.Errorf("ScanStruct: duplicate struct field name or tag %q in %s", name, v.Type())
		}
		outFieldPtrs[name] = v.Field(i).Addr().Interface()
	}
	return nil
}

// structFieldValues returns the struct field names using the passed namer ignoring names in ignoreNames
// and if restrictToNames is not empty, then filtering out names not in it.
// struct fields with ,readonly suffix in their struct field naming tag will not be returned
// because this function is intended for getting struct values for writing.
// If true is passed for keepPK, then ignoreNames and restrictToNames are not applied to names with
// the ,pk suffix in their struct field naming tag.
// The same number of pkCol bools will be returend as names, every corresponding bool marking
// if the name had the ,pk suffix in their struct field naming tag.
// If false is passed for keepReadOnly then
func structFieldValues(v reflect.Value, namer sqldb.StructFieldNamer, ignoreNames, restrictToNames []string, keepPK bool) (names []string, flags []sqldb.FieldFlag, vals []any) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		name, flag, ok := namer.StructFieldName(field)
		if !ok || flag.IsReadOnly() {
			continue
		}

		if field.Anonymous {
			embedNames, embedFlags, embedValues := structFieldValues(v.Field(i), namer, ignoreNames, restrictToNames, keepPK)
			names = append(names, embedNames...)
			flags = append(flags, embedFlags...)
			vals = append(vals, embedValues...)
			continue
		}

		if validName(name, ignoreNames, restrictToNames) || (flag.IsPrimaryKey() && keepPK && validName(name, nil, nil)) {
			names = append(names, name)
			flags = append(flags, flag)
			vals = append(vals, v.Field(i).Interface())
		}
	}
	return names, flags, vals
}

// validName returns if a name not empty and not in ignoreNames
// and if restrictToNames is not empty, then not in restrictToNames.
func validName(name string, ignoreNames, restrictToNames []string) bool {
	if name == "" {
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

func GetStructFieldIndices(t reflect.Type, namer sqldb.StructFieldNamer) (fieldIndices map[string][]int, err error) {
	fieldIndices = make(map[string][]int)
	err = getStructFieldIndices(t, namer, nil, fieldIndices)
	if err != nil {
		return nil, err
	}
	return fieldIndices, nil
}

func getStructFieldIndices(structType reflect.Type, namer sqldb.StructFieldNamer, parentIndices []int, outFieldIndices map[string][]int) error {
	t := structType
	if t.Kind() == reflect.Ptr {
		t = structType.Elem()
	}
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("struct or pointer to struct type expected, but got %s", structType)
	}

	numFields := t.NumField()
	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		name, _, ok := namer.StructFieldName(field)
		if !ok {
			continue
		}

		if field.Anonymous {
			err := getStructFieldIndices(field.Type, namer, append(parentIndices, field.Index...), outFieldIndices)
			if err != nil {
				return err
			}
			continue
		}

		if outFieldIndices[name] != nil {
			return fmt.Errorf("ScanStruct: duplicate struct field name or tag %q in %s", name, t)
		}
		outFieldIndices[name] = append(parentIndices, field.Index...)
	}

	return nil
}
