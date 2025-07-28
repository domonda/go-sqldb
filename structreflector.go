package sqldb

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/domonda/go-types/strutil"
)

// StructReflector is used to map struct type fields to column names
// and indicate special column properies via flags.
type StructReflector interface {
	// TableNameForStruct returns the table name for a struct type
	TableNameForStruct(t reflect.Type) (table string, err error)

	// MapStructField returns the Column information for a reflected struct field
	// If false is returned for use then the field is not mapped.
	// An empty name and true for use indicates an embedded struct
	// field whose fields should be recursively mapped.
	MapStructField(field reflect.StructField) (column ColumnInfo, use bool)
}

// NewTaggedStructReflector returns a default mapping.
func NewTaggedStructReflector() *TaggedStructReflector {
	return &TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "primarykey",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: IgnoreStructField,
	}
}

// TaggedStructReflector implements StructReflector
var _ StructReflector = new(TaggedStructReflector)

// TaggedStructReflector implements StructReflector with a struct field NameTag
// to be used for naming and a UntaggedNameFunc in case the NameTag is not set.
type TaggedStructReflector struct {
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

func (m *TaggedStructReflector) TableNameForStruct(t reflect.Type) (table string, err error) {
	return TableNameForStruct(t, m.NameTag)
}

func (m *TaggedStructReflector) MapStructField(field reflect.StructField) (column ColumnInfo, use bool) {
	if field.Anonymous {
		tag, hasTag := field.Tag.Lookup(m.NameTag)
		if !hasTag {
			// Embedded struct fields are ok if not tagged with IgnoreName
			return ColumnInfo{}, true
		}
		columnName, _, _ := strings.Cut(tag, ",")
		columnName = strings.TrimSpace(columnName)
		// Embedded struct fields are ok if not tagged with IgnoreName
		return ColumnInfo{}, columnName != m.Ignore
	}
	if !field.IsExported() {
		// Not exported struct fields that are not
		// anonymously embedded structs are not ok
		return ColumnInfo{}, false
	}

	if tag, hasTag := field.Tag.Lookup(m.NameTag); hasTag {
		column.Name, tag, _ = strings.Cut(tag, ",")
		column.Name = strings.TrimSpace(column.Name)

		str, tag, ok := strings.Cut(tag, ",")
		for str != "" || ok {
			switch strings.TrimSpace(str) {
			case m.PrimaryKey:
				column.PrimaryKey = true
			case m.ReadOnly:
				column.ReadOnly = true
			case m.Default:
				column.HasDefault = true
			}
			str, tag, ok = strings.Cut(tag, ",")
		}
	} else {
		column.Name = m.UntaggedNameFunc(field.Name)
	}

	if column.Name == "" || column.Name == m.Ignore {
		return ColumnInfo{}, false
	}
	return column, true
}

func (n TaggedStructReflector) String() string {
	return fmt.Sprintf("NameTag: %q", n.NameTag)
}

// IgnoreStructField can be used as TaggedStructReflector.UntaggedNameFunc
// to ignore fields that don't have TaggedStructReflector.NameTag.
func IgnoreStructField(string) string { return "" }

// ToSnakeCase converts s to snake case
// by lower casing everything and inserting '_'
// before every new upper case character in s.
// Whitespace, symbol, and punctuation characters
// will be replace by '_'.
var ToSnakeCase = strutil.ToSnakeCase
