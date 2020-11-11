package sqldb

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// StructFieldNamer is used to map struct type fields to column names
// and indicate if the column is a primary key.
type StructFieldNamer interface {
	// StructFieldName returns the column name for reflected struct field
	// and if the column is a primary key (pk).
	// A name is returned for every ok field, non ok fields will be ignored.
	StructFieldName(field reflect.StructField) (name string, pk, ok bool)
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

func (n StructFieldTagNaming) StructFieldName(field reflect.StructField) (name string, pk, ok bool) {
	if field.Anonymous || (field.Type.Kind() == reflect.Struct && field.Type.Name() == "") {
		// Either an embedded struct or inline declared struct type
		name, hasTag := field.Tag.Lookup(n.NameTag)
		if !hasTag {
			// Embedded struct fields are ok if not tagged with IgnoreName
			return "", false, true
		}
		if i := strings.IndexByte(name, ','); i != -1 {
			name = name[:i]
		}
		// Embedded struct fields are ok if not tagged with IgnoreName
		return "", false, name != n.IgnoreName
	}
	if field.PkgPath != "" {
		// Not exported struct fields that are not
		// anonymously embedded structs are not ok
		return "", false, false
	}

	name, hasTag := field.Tag.Lookup(n.NameTag)
	if hasTag {
		if i := strings.IndexByte(name, ','); i != -1 {
			pk = name[i+1:] == "pk"
			name = name[:i]
		}
	} else {
		name = n.UntaggedNameFunc(field.Name)
	}

	if name == "" || name == n.IgnoreName {
		return "", false, false
	}
	return name, pk, true
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
