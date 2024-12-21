package db

import (
	"fmt"
	"reflect"
)

// StructWithTableName is a marker interface for structs
// that embed a TableName field to specify the table name.
type StructWithTableName interface {
	HasTableName()
}

// TableName implements the StructWithTableName marker interface
var _ StructWithTableName = TableName{}

// TableName is an empty struct that can be embedded in other structs
// to specify the table name for the struct using a struct tag.
type TableName struct{}

// HasTableName implements the StructWithTableName interface
func (TableName) HasTableName() {}

func TableNameForStruct(t reflect.Type, tagKey string) (table string, err error) {
	structType := t
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return "", fmt.Errorf("db.StructTable: %s is not a struct or pointer to a struct", t)
	}
	if tagKey == "" {
		return "", fmt.Errorf("db.StructTable: tagKey is empty")
	}
	tableNameType := reflect.TypeFor[TableName]()
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.Anonymous && field.Type == tableNameType {
			table = field.Tag.Get(tagKey)
			if table == "" {
				return "", fmt.Errorf("db.StructTable: embedded db.Table has no tag '%s'", tagKey)
			}
			return table, nil
		}
	}
	return "", fmt.Errorf("db.StructTable: struct type %s has no embedded db.Table field", t)
}
