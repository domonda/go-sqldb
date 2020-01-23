package sqldb

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

type StructFieldNamer interface {
	StructFieldName(field reflect.StructField) (name string, pk bool)
}

type StructFieldNamerFunc func(field reflect.StructField) (name string, pk bool)

func (f StructFieldNamerFunc) StructFieldName(field reflect.StructField) (name string, pk bool) {
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
