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
	StructFieldName(field reflect.StructField) (name string, pk bool)
}

// StructFieldNamerFunc implements the StructFieldNamer interface with a function
type StructFieldNamerFunc func(field reflect.StructField) (name string, pk bool)

func (f StructFieldNamerFunc) StructFieldName(field reflect.StructField) (name string, pk bool) {
	return f(field)
}

// DefaultStructFieldTagNaming provides the default StructFieldTagNaming
// using "db" as NameTag and ToSnakeCase as UntaggedNameFunc.
// Implements StructFieldNamer.
var DefaultStructFieldTagNaming = StructFieldTagNaming{
	NameTag:          "db",
	UntaggedNameFunc: ToSnakeCase,
}

// StructFieldTagNaming implements StructFieldNamer with a struct field NameTag
// to be used for naming and a UntaggedNameFunc in case the NameTag is not set.
type StructFieldTagNaming struct {
	_Named_Fields_Required struct{}

	// NameTag is the struct field tag to be used as column name
	NameTag string
	// UntaggedNameFunc will be called with the struct field name to
	// return a column name in case the struct field has no tag named NameTag.
	UntaggedNameFunc func(fieldName string) string
}

func (n StructFieldTagNaming) StructFieldName(field reflect.StructField) (name string, pk bool) {
	if tag, ok := field.Tag.Lookup(n.NameTag); ok && tag != "" {
		if i := strings.IndexByte(tag, ','); i != -1 {
			pk = tag[i+1:] == "pk"
			name = tag[:i]
			if name != "" {
				return name, pk
			}
		} else {
			return tag, false
		}
	}
	if n.UntaggedNameFunc == nil {
		return field.Name, pk
	}
	return n.UntaggedNameFunc(field.Name), pk
}

func (n StructFieldTagNaming) String() string {
	return fmt.Sprintf("NameTag: %q", n.NameTag)
}

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
