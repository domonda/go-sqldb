package db

import (
	"fmt"
	"reflect"
)

type Table struct{}

var typeOfTable = reflect.TypeFor[Table]()

func TableForStruct(t reflect.Type, tagKey string) (table string, err error) {
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("db.StructTable: %s is not a struct", t)
	}
	if tagKey == "" {
		return "", fmt.Errorf("db.StructTable: tagKey is empty")
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == typeOfTable {
			table = field.Tag.Get(tagKey)
			if table == "" {
				return "", fmt.Errorf("db.StructTable: embedded db.Table has no tag '%s'", tagKey)
			}
			return table, nil
		}
	}
	return "", fmt.Errorf("db.StructTable: struct type %s has no embedded db.Table field", t)
}
