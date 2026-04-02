package sqldb

import (
	"fmt"
	"reflect"
)

// StructWithTableName is a marker interface for structs
// that embed a TableName field to specify the table name.
type StructWithTableName interface {
	HasTableName() // Interface marker method
}

// TableName implements the StructWithTableName marker interface
var _ StructWithTableName = TableName{}

// TableName is an empty struct that implements StructWithTableName
// and is intended to be embedded in other structs
// to specify the table name for the struct using a struct tag.
//
// Example:
//
//	type MyTable struct {
//	    TableName `db:"my_table"`
//	}
type TableName struct{}

// HasTableName implements the StructWithTableName interface
func (TableName) HasTableName() {}

// TableNameForStruct returns the table name for a struct type
// by looking for an embedded [TableName] field with the given tagKey.
// It searches recursively through anonymous embedded struct fields.
func TableNameForStruct(t reflect.Type, tagKey string) (table string, err error) {
	structType := t
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return "", fmt.Errorf("TableNameForStruct: %s is not a struct or pointer to a struct", t)
	}
	if tagKey == "" {
		return "", fmt.Errorf("TableNameForStruct: tagKey is empty")
	}
	table, found, err := tableNameForStructRecursive(structType, tagKey)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("TableNameForStruct: struct type %s has no embedded TableName field", t)
	}
	return table, nil
}

func tableNameForStructRecursive(structType reflect.Type, tagKey string) (table string, found bool, err error) {
	tableNameType := reflect.TypeFor[TableName]()
	for i := range structType.NumField() {
		field := structType.Field(i)
		if !field.Anonymous {
			continue
		}
		if field.Type == tableNameType {
			table = field.Tag.Get(tagKey)
			if table == "" {
				return "", false, fmt.Errorf("TableNameForStruct: embedded TableName has no tag %q", tagKey)
			}
			return table, true, nil
		}
		// Recurse into anonymous embedded struct fields
		fieldType := field.Type
		for fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			table, found, err = tableNameForStructRecursive(fieldType, tagKey)
			if found || err != nil {
				return table, found, err
			}
		}
	}
	return "", false, nil
}
