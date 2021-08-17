package sqldb

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// FieldFlag is a bitmask for special properties
// of how struct fields relate to database columns.
type FieldFlag uint

const (
	FieldFlagPrimaryKey FieldFlag = 1 << iota
	FieldFlagReadOnly
)

func (f FieldFlag) IsPrimaryKey() bool { return f&FieldFlagPrimaryKey != 0 }
func (f FieldFlag) IsReadOnly() bool   { return f&FieldFlagReadOnly != 0 }

// StructFieldNamer is used to map struct type fields to column names
// and indicate special column properies via flags.
type StructFieldNamer interface {
	// StructFieldName returns the column name for reflected struct field
	// and flags for special column properies.
	// If the struct field can't be mapped, false is returned for ok.
	StructFieldName(field reflect.StructField) (name string, flags FieldFlag, ok bool)
}

// DefaultStructFieldTagNaming provides the default StructFieldTagNaming
// using "db" as NameTag and IgnoreStructField as UntaggedNameFunc.
// Implements StructFieldNamer.
var DefaultStructFieldTagNaming = StructFieldTagNaming{
	NameTag:          "db",
	IgnoreName:       "-",
	UntaggedNameFunc: IgnoreStructField,
}

// StructFieldTagNaming implements StructFieldNamer with a struct field NameTag
// to be used for naming and a UntaggedNameFunc in case the NameTag is not set.
type StructFieldTagNaming struct {
	_Named_Fields_Required struct{}

	// NameTag is the struct field tag to be used as column name
	NameTag string

	// IgnoreName will cause a struct field to be ignored if it has that name
	IgnoreName string

	// UntaggedNameFunc will be called with the struct field name to
	// return a column name in case the struct field has no tag named NameTag.
	UntaggedNameFunc func(fieldName string) string
}

func (n StructFieldTagNaming) StructFieldName(field reflect.StructField) (name string, flags FieldFlag, ok bool) {
	if field.Anonymous {
		name, hasTag := field.Tag.Lookup(n.NameTag)
		if !hasTag {
			// Embedded struct fields are ok if not tagged with IgnoreName
			return "", 0, true
		}
		if i := strings.IndexByte(name, ','); i != -1 {
			name = name[:i]
		}
		// Embedded struct fields are ok if not tagged with IgnoreName
		return "", 0, name != n.IgnoreName
	}
	if !field.IsExported() {
		// Not exported struct fields that are not
		// anonymously embedded structs are not ok
		return "", 0, false
	}

	tag, hasTag := field.Tag.Lookup(n.NameTag)
	if hasTag {
		for i, part := range strings.Split(tag, ",") {
			// First part is the name
			if i == 0 {
				name = part
				continue
			}
			// Follow on parts are flags
			switch part {
			case "pk":
				flags |= FieldFlagPrimaryKey
			case "readonly":
				flags |= FieldFlagReadOnly
			}
		}
	} else {
		name = n.UntaggedNameFunc(field.Name)
	}

	if name == "" || name == n.IgnoreName {
		return "", 0, false
	}
	return name, flags, true
}

func (n StructFieldTagNaming) String() string {
	return fmt.Sprintf("NameTag: %q", n.NameTag)
}

// IgnoreStructField can be used as StructFieldTagNaming.UntaggedNameFunc
// to ignore fields that don't have StructFieldTagNaming.NameTag.
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
