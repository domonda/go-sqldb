package sqldb

import (
	"reflect"
	"strings"
	"unicode"
)

type StructFieldNamer interface {
	StructFieldName(field reflect.StructField) string
}

type StructFieldNamerFunc func(field reflect.StructField) string

func (f StructFieldNamerFunc) StructFieldName(field reflect.StructField) string {
	return f(field)
}

var DefaultStructFieldTagNaming = StructFieldTagNaming{
	NameTag:          "db",
	UntaggedNameFunc: ToSnakeCase,
}

type StructFieldTagNaming struct {
	NameTag          string
	UntaggedNameFunc func(string) string
}

func (n StructFieldTagNaming) StructFieldName(field reflect.StructField) string {
	if tag, ok := field.Tag.Lookup(n.NameTag); ok && tag != "" {
		if i := strings.IndexByte(tag, ','); i != -1 {
			return tag[:i]
		}
		return tag
	}
	if n.UntaggedNameFunc == nil {
		return field.Name
	}
	return n.UntaggedNameFunc(field.Name)
}

// ToSnakeCase converts s to snake case
// by lower casing everything and inserting '_'
// before every new upper case character in s.
func ToSnakeCase(s string) string {
	b := strings.Builder{}
	b.Grow(len(s))
	lastWasUpper := true
	for _, c := range s {
		l := unicode.ToLower(c)
		isUpper := l != c
		if isUpper && !lastWasUpper {
			b.WriteByte('_')
		}
		b.WriteRune(l)
		lastWasUpper = isUpper
	}
	return b.String()
}
