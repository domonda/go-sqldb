package reflection

import (
	"fmt"
	"reflect"
	"strings"
)

// TaggedStructMapper implements StructFieldMapper with a struct field NameTag
// to be used for naming and a UntaggedNameFunc in case the NameTag is not set.
type TaggedStructMapper struct {
	_Named_Fields_Required struct{}

	// NameTag is the struct field tag to be used as column name
	NameTag string

	// Ignore will cause a struct field to be ignored if it has that name
	Ignore string

	PrimaryKey string
	ReadOnly   string
	Default    string

	// UntaggedNameFunc will be called with the struct field name to
	// return a column name in case the struct field has no tag named NameTag.
	UntaggedNameFunc func(fieldName string) string
}

// NewTaggedStructMapper returns a default mapping.
func NewTaggedStructMapper() *TaggedStructMapper {
	return &TaggedStructMapper{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: IgnoreStructField,
	}
}

func (m *TaggedStructMapper) ReflectStructMapping(structType reflect.Type) (*StructMapping, error) {
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("passed type %s is not a struct", structType)
	}
	mapping := &StructMapping{
		StructType: structType,
		ColumnMap:  make(map[string]*StructColumn),
	}
	err := m.reflectStructMapping(structType, mapping)
	if err != nil {
		return nil, err
	}
	return mapping, nil
}

func (m *TaggedStructMapper) reflectStructMapping(structType reflect.Type, mapping *StructMapping) error {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldTable, name, flags, use := m.mapStructField(field)
		if !use {
			continue
		}

		if name == "" {
			// Embedded struct field
			err := m.reflectStructMapping(field.Type, mapping)
			if err != nil {
				return err
			}
			continue
		}

		if fieldTable != "" && fieldTable != mapping.Table {
			if mapping.Table != "" {
				return fmt.Errorf("table name not unique (%s vs %s) in struct %s", mapping.Table, fieldTable, mapping.StructType)
			}
			mapping.Table = fieldTable
		}

		column := &StructColumn{
			Name:       name,
			Flags:      flags,
			FieldIndex: field.Index,
			FieldType:  field,
		}
		mapping.Columns = append(mapping.Columns, column)
		mapping.ColumnMap[name] = column
	}
	return nil
}

func (m *TaggedStructMapper) mapStructField(field reflect.StructField) (table, column string, flags StructFieldFlags, use bool) {
	if field.Anonymous {
		column, hasTag := field.Tag.Lookup(m.NameTag)
		if !hasTag {
			// Embedded struct fields are ok if not tagged with IgnoreName
			return "", "", 0, true
		}
		if i := strings.IndexByte(column, ','); i != -1 {
			column = column[:i]
		}
		// Embedded struct fields are ok if not tagged with IgnoreName
		return "", "", 0, column != m.Ignore
	}

	if !field.IsExported() {
		// Not exported struct fields that are not
		// anonymously embedded structs are not ok
		return "", "", 0, false
	}

	tag, hasTag := field.Tag.Lookup(m.NameTag)
	if hasTag {
		for i, part := range strings.Split(tag, ",") {
			// First part is the name
			if i == 0 {
				column = part
				continue
			}
			// Follow on parts are flags
			flag, value, _ := strings.Cut(part, "=")
			switch flag {
			case "":
				// Ignore empty flags
			case m.PrimaryKey:
				flags |= FlagPrimaryKey
				table = value
			case m.ReadOnly:
				flags |= FlagReadOnly
			case m.Default:
				flags |= FlagHasDefault
			}
		}
	} else if m.UntaggedNameFunc != nil {
		column = m.UntaggedNameFunc(field.Name)
	}

	if column == m.Ignore || column == "" {
		return "", "", 0, false
	}
	return table, column, flags, true
}
