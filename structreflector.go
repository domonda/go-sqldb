package sqldb

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// StructReflector is used to map struct type fields to column names
// and indicate special column properties via flags.
type StructReflector interface {
	// TableNameForStruct returns the table name for a struct type
	TableNameForStruct(t reflect.Type) (table string, err error)

	// MapStructField returns the Column information for a reflected struct field
	// If false is returned for use then the field is not mapped.
	// An empty name and true for use indicates an embedded struct
	// field whose fields should be recursively mapped.
	MapStructField(field reflect.StructField) (column ColumnInfo, use bool)

	// ColumnPointers returns addressable pointers to the struct fields
	// corresponding to the given query result column names,
	// suitable for use with Rows.Scan.
	ColumnPointers(structVal reflect.Value, columns []string) (pointers []any, err error)
}

// NewTaggedStructReflector returns a default mapping.
func NewTaggedStructReflector() *TaggedStructReflector {
	return &TaggedStructReflector{
		NameTag:                    "db",
		Ignore:                     "-",
		PrimaryKey:                 "primarykey",
		ReadOnly:                   "readonly",
		Default:                    "default",
		UntaggedNameFunc:           IgnoreStructField,
		FailOnUnmappedColumns:      false,
		FailOnUnmappedStructFields: false,
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

	// FailOnUnmappedColumns controls whether ColumnPointers returns an error
	// when query result columns have no mapped struct fields.
	// If false (the default), unmapped columns are silently ignored
	// by using discard scan destinations.
	FailOnUnmappedColumns bool

	// FailOnUnmappedStructFields controls whether ColumnPointers returns an error
	// when the struct has mapped fields with no corresponding query result column.
	// If false (the default), unmapped struct fields are silently left unchanged.
	FailOnUnmappedStructFields bool
}

// TableNameForStruct implements StructReflector.TableNameForStruct.
func (m *TaggedStructReflector) TableNameForStruct(t reflect.Type) (table string, err error) {
	return TableNameForStruct(t, m.NameTag)
}

// MapStructField implements StructReflector.MapStructField.
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

// ColumnPointers implements StructReflector.ColumnPointers.
func (m *TaggedStructReflector) ColumnPointers(structVal reflect.Value, columns []string) (pointers []any, err error) {
	if len(columns) == 0 {
		return nil, errors.New("no columns")
	}
	rs, err := reflectStruct(m, structVal.Type())
	if err != nil {
		return nil, err
	}
	pointers = make([]any, len(columns))
	for i, col := range columns {
		idx, ok := rs.ColumnIndex[col]
		if !ok {
			continue
		}
		pointers[i] = structVal.FieldByIndex(rs.Fields[idx].FieldIndex).Addr().Interface()
	}
	for i, ptr := range pointers {
		if ptr != nil {
			continue
		}
		if !m.FailOnUnmappedColumns {
			// Use discard destination for unmapped columns
			pointers[i] = new(any)
			continue
		}
		nilCols := new(strings.Builder)
		for j, ptr := range pointers {
			if ptr != nil {
				continue
			}
			if nilCols.Len() > 0 {
				nilCols.WriteString(", ")
			}
			fmt.Fprintf(nilCols, "column=%s, index=%d", columns[j], j)
		}
		return nil, fmt.Errorf("columns have no mapped struct fields in %s: %s", structVal.Type(), nilCols)
	}
	if m.FailOnUnmappedStructFields {
		columnSet := make(map[string]struct{}, len(columns))
		for _, col := range columns {
			columnSet[col] = struct{}{}
		}
		var unmapped *strings.Builder
		for _, f := range rs.Fields {
			if _, ok := columnSet[f.Column.Name]; !ok {
				if unmapped == nil {
					unmapped = new(strings.Builder)
				}
				if unmapped.Len() > 0 {
					unmapped.WriteString(", ")
				}
				unmapped.WriteString(f.Column.Name)
			}
		}
		if unmapped != nil {
			return nil, fmt.Errorf("struct fields have no mapped columns in query result for %s: %s", structVal.Type(), unmapped)
		}
	}
	return pointers, nil
}

func (n TaggedStructReflector) String() string {
	return fmt.Sprintf("NameTag: %q", n.NameTag)
}

// IgnoreStructField can be used as TaggedStructReflector.UntaggedNameFunc
// to ignore fields that don't have TaggedStructReflector.NameTag.
func IgnoreStructField(string) string { return "" }

// ToSnakeCase is defined in snakecase.go
