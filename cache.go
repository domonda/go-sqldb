package sqldb

import (
	"fmt"
	"reflect"
	"slices"
	"sync"
)

// reflectedStructField holds the cached reflection data
// for a single flattened struct field mapped to a database column.
type reflectedStructField struct {
	Column      ColumnInfo
	StructField reflect.StructField
	FieldIndex  []int // multi-level index for reflect.Value.FieldByIndex
}

// reflectedStruct holds the cached reflection data
// for a struct type flattened across all embedded structs.
type reflectedStruct struct {
	Fields      []reflectedStructField
	ColumnIndex map[string]int // column name -> index in Fields
}

type queryCache struct {
	query              string
	structFieldIndices [][]int
}

type queryRowByPKCacheEntry struct {
	query        string
	numPKColumns int
}

var (
	reflectedStructCache    = make(map[reflect.Type]map[StructReflector]*reflectedStruct)
	reflectedStructCacheMtx sync.RWMutex

	insertRowStructQueryCache    = make(map[reflect.Type]map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryCache)
	insertRowStructQueryCacheMtx sync.RWMutex

	upsertRowStructQueryCache    = make(map[reflect.Type]map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryCache)
	upsertRowStructQueryCacheMtx sync.RWMutex

	deleteRowStructQueryCache    = make(map[reflect.Type]map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryCache)
	deleteRowStructQueryCacheMtx sync.RWMutex

	updateRowStructQueryCache    = make(map[reflect.Type]map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryCache)
	updateRowStructQueryCacheMtx sync.RWMutex

	queryRowByPKCache    = make(map[reflect.Type]map[StructReflector]map[QueryBuilder]map[QueryFormatter]queryRowByPKCacheEntry)
	queryRowByPKCacheMtx sync.RWMutex
)

// reflectStruct returns the cached reflected struct data
// for the given reflector and struct type.
// It uses a read-lock fast path and a write-lock slow path.
func reflectStruct(reflector StructReflector, structType reflect.Type) (*reflectedStruct, error) {
	reflectedStructCacheMtx.RLock()
	if byReflector, ok := reflectedStructCache[structType]; ok {
		if rs, ok := byReflector[reflector]; ok {
			reflectedStructCacheMtx.RUnlock()
			return rs, nil
		}
	}
	reflectedStructCacheMtx.RUnlock()

	// Cache miss — build and store
	fields := flattenStructFields(reflector, structType, nil)
	columnIndex := make(map[string]int, len(fields))
	for i, f := range fields {
		if _, exists := columnIndex[f.Column.Name]; exists {
			return nil, fmt.Errorf("duplicate column %q in struct %s", f.Column.Name, structType)
		}
		columnIndex[f.Column.Name] = i
	}
	rs := &reflectedStruct{
		Fields:      fields,
		ColumnIndex: columnIndex,
	}

	// Could have already been created and stored by another goroutine
	// but result is the same
	reflectedStructCacheMtx.Lock()
	if _, ok := reflectedStructCache[structType]; !ok {
		reflectedStructCache[structType] = make(map[StructReflector]*reflectedStruct)
	}
	reflectedStructCache[structType][reflector] = rs
	reflectedStructCacheMtx.Unlock()

	return rs, nil
}

// flattenStructFields recursively traverses the struct type
// and returns a flat list of all mapped fields with their
// full field indices for FieldByIndex access.
func flattenStructFields(reflector StructReflector, structType reflect.Type, parentIndex []int) []reflectedStructField {
	var fields []reflectedStructField
	for i := range structType.NumField() {
		field := structType.Field(i)
		column, use := reflector.MapStructField(field)
		if !use {
			continue
		}
		if column.IsEmbeddedField() {
			// Recurse into embedded struct
			embeddedFields := flattenStructFields(
				reflector,
				field.Type,
				slices.Concat(parentIndex, field.Index),
			)
			fields = append(fields, embeddedFields...)
			continue
		}
		fields = append(fields, reflectedStructField{
			Column:      column,
			StructField: field,
			FieldIndex:  slices.Concat(parentIndex, field.Index),
		})
	}
	return fields
}

// ClearQueryCaches clears all internal query caches.
// This is useful for testing and debugging to ensure
// that queries are rebuilt from scratch.
func ClearQueryCaches() {
	reflectedStructCacheMtx.Lock()
	clear(reflectedStructCache)
	reflectedStructCacheMtx.Unlock()

	insertRowStructQueryCacheMtx.Lock()
	clear(insertRowStructQueryCache)
	insertRowStructQueryCacheMtx.Unlock()

	upsertRowStructQueryCacheMtx.Lock()
	clear(upsertRowStructQueryCache)
	upsertRowStructQueryCacheMtx.Unlock()

	deleteRowStructQueryCacheMtx.Lock()
	clear(deleteRowStructQueryCache)
	deleteRowStructQueryCacheMtx.Unlock()

	updateRowStructQueryCacheMtx.Lock()
	clear(updateRowStructQueryCache)
	updateRowStructQueryCacheMtx.Unlock()

	queryRowByPKCacheMtx.Lock()
	clear(queryRowByPKCache)
	queryRowByPKCacheMtx.Unlock()
}
