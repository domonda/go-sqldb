package reflection

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// FieldFlag is a bitmask for special properties
// of how struct fields relate to database columns.
type FieldFlag uint

// PrimaryKey indicates if FieldFlagPrimaryKey is set
func (f FieldFlag) PrimaryKey() bool { return f&FieldFlagPrimaryKey != 0 }

// ReadOnly indicates if FieldFlagReadOnly is set
func (f FieldFlag) ReadOnly() bool { return f&FieldFlagReadOnly != 0 }

// Default indicates if FieldFlagDefault is set
func (f FieldFlag) Default() bool { return f&FieldFlagDefault != 0 }

const (
	// FieldFlagPrimaryKey marks a field as primary key
	FieldFlagPrimaryKey FieldFlag = 1 << iota

	// FieldFlagReadOnly marks a field as read-only
	FieldFlagReadOnly

	// FieldFlagDefault marks a field as having a column default value
	FieldFlagDefault
)

// StructFieldMapper is used to map struct type fields to column names
// and indicate special column properies via flags.
type StructFieldMapper interface {
	// MapStructField returns the column name for a reflected struct field
	// and flags for special column properies.
	// If false is returned for use then the field is not mapped.
	// An empty name and true for use indicates an embedded struct
	// field whose fields should be recursively mapped.
	MapStructField(field reflect.StructField) (table, column string, flags FieldFlag, use bool)
}

// NewTaggedStructFieldMapping returns a default mapping.
func NewTaggedStructFieldMapping() *TaggedStructFieldMapping {
	return &TaggedStructFieldMapping{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: IgnoreStructField,
	}
}

// TaggedStructFieldMapping implements StructFieldMapper with a struct field NameTag
// to be used for naming and a UntaggedNameFunc in case the NameTag is not set.
type TaggedStructFieldMapping struct {
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

func (m *TaggedStructFieldMapping) MapStructField(field reflect.StructField) (table, column string, flags FieldFlag, use bool) {
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
				flags |= FieldFlagPrimaryKey
				table = value
			case m.ReadOnly:
				flags |= FieldFlagReadOnly
			case m.Default:
				flags |= FieldFlagDefault
			}
		}
	} else {
		column = m.UntaggedNameFunc(field.Name)
	}

	if column == "" || column == m.Ignore {
		return "", "", 0, false
	}
	return table, column, flags, true
}

func (n TaggedStructFieldMapping) String() string {
	return fmt.Sprintf("NameTag: %q", n.NameTag)
}

// IgnoreStructField can be used as TaggedStructFieldMapping.UntaggedNameFunc
// to ignore fields that don't have TaggedStructFieldMapping.NameTag.
func IgnoreStructField(string) string { return "" }

// ToSnakeCase converts s to snake case
// by lower casing everything and inserting '_'
// before every new upper case character in s.
func ToSnakeCase(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	lastWasUpper := true
	for _, r := range s {
		lr := unicode.ToLower(r)
		isUpper := lr != r
		if isUpper && !lastWasUpper {
			b.WriteByte('_')
		}
		b.WriteRune(lr)
		lastWasUpper = isUpper
	}
	return b.String()
}
