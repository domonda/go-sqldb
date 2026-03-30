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
	tableNameType := reflect.TypeFor[TableName]()
	for i := range structType.NumField() {
		field := structType.Field(i)
		if field.Anonymous && field.Type == tableNameType {
			table = field.Tag.Get(tagKey)
			if table == "" {
				return "", fmt.Errorf("TableNameForStruct: embedded TableName has no tag %q", tagKey)
			}
			return table, nil
		}
	}
	return "", fmt.Errorf("TableNameForStruct: struct type %s has no embedded TableName field", t)
}
