package reflection

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type StructMapping struct {
	StructType reflect.Type
	Table      string
	Columns    []*StructColumn
	ColumnMap  map[string]*StructColumn
}

type StructColumn struct {
	Name       string
	Flags      StructFieldFlags
	FieldIndex []int
	FieldType  reflect.StructField
}

type mappingKey struct {
	reflect.Type
	StructMapper
}

var (
	cachedMappings    = make(map[mappingKey]*StructMapping)
	cachedMappingsMtx sync.Mutex
)

func CachedStructMapping(t reflect.Type, m StructMapper) (*StructMapping, error) {
	cachedMappingsMtx.Lock()
	defer cachedMappingsMtx.Unlock()

	key := mappingKey{t, m}

	if mapping, ok := cachedMappings[key]; ok {
		return mapping, nil
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("passed type %s is not a struct", t)
	}
	if m == nil {
		return nil, errors.New("passed nil StructMapper")
	}
	mapping, err := m.ReflectStructMapping(t)
	if err != nil {
		return nil, err
	}
	cachedMappings[key] = mapping
	return mapping, nil
}

func (m *StructMapping) StructColumnValues(strct any, filter ColumnFilter) ([]any, error) {
	v := reflect.ValueOf(strct)
	switch v.Kind() {
	case reflect.Struct:
		// ok
	case reflect.Pointer:
		if v.IsNil() {
			return nil, fmt.Errorf("passed nil %T", strct)
		}
		v = v.Elem()
		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("passed type %T is not a struct pointer", strct)
		}
	default:
		return nil, fmt.Errorf("passed type %T is not a struct or struct pointer", strct)
	}
	if v.Type() != m.StructType {
		return nil, fmt.Errorf("passed struct of type %s to %s StructMapping", v.Type(), m.StructType)
	}

	if filter == nil {
		vals := make([]any, len(m.Columns))
		for i, col := range m.Columns {
			vals[i] = v.FieldByIndex(col.FieldIndex).Interface()
		}
		return vals, nil
	}

	vals := make([]any, 0, len(m.Columns))
	for _, col := range m.Columns {
		val := v.FieldByIndex(col.FieldIndex)
		if !filter.IgnoreColumn(col, val) {
			vals = append(vals, val.Interface())
		}
	}
	return vals, nil
}

// func (m *StructMapping) StructColumnPointers(structPtr any, filter ColumnFilter) ([]any, error) {
// 	v := reflect.ValueOf(structPtr)
// 	if v.Kind() != reflect.Pointer {
// 		return nil, fmt.Errorf("passed type %T is not a struct pointer", structPtr)
// 	}
// 	if v.IsNil() {
// 		return nil, fmt.Errorf("passed nil %T", structPtr)
// 	}
// 	v = v.Elem()
// 	if v.Kind() != reflect.Struct {
// 		return nil, fmt.Errorf("passed type %T is not a struct pointer", structPtr)
// 	}
// 	if v.Type() != m.StructType {
// 		return nil, fmt.Errorf("passed struct of type %s to %s StructMapping", v.Type(), m.StructType)
// 	}

// 	if filter == nil {
// 		vals := make([]any, len(m.Columns))
// 		for i, col := range m.Columns {
// 			vals[i] = v.FieldByIndex(col.FieldIndex).Addr().Interface()
// 		}
// 		return vals, nil
// 	}

// 	vals := make([]any, 0, len(m.Columns))
// 	for _, col := range m.Columns {
// 		val := v.FieldByIndex(col.FieldIndex)
// 		if !filter.IgnoreColumn(col, val) {
// 			vals = append(vals, val.Addr().Interface())
// 		}
// 	}
// 	return vals, nil
// }

// ScanStruct scans values of a srcRow into a destStruct which must be passed as pointer.
func (m *StructMapping) ScanStruct(srcRow Row, structPtr any, filter ColumnFilter) error {
	v := reflect.ValueOf(structPtr)
	// if v.Kind() != reflect.Pointer {
	// 	return fmt.Errorf("passed type %T is not a struct pointer", structPtr)
	// }
	// if v.IsNil() {
	// 	return fmt.Errorf("passed nil %T", structPtr)
	// }
	// v = v.Elem()
	// if v.Kind() != reflect.Struct {
	// 	return fmt.Errorf("passed type %T is not a struct pointer", structPtr)
	// }
	// if v.Type() != m.StructType {
	// 	return fmt.Errorf("passed struct of type %s to %s StructMapping", v.Type(), m.StructType)
	// }

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
		return fmt.Errorf("passed type %T is not a struct pointer", structPtr)
	}

	columns, err := srcRow.Columns()
	if err != nil {
		return err
	}

	fieldPointers := make([]any, len(columns))
	for i, name := range columns {
		col, ok := m.ColumnMap[name]
		if !ok {
			return fmt.Errorf("no mapping for column %s to struct %s", name, m.StructType)
		}
		fieldPointers[i] = v.FieldByIndex(col.FieldIndex).Addr().Interface()
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
