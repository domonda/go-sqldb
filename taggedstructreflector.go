package sqldb

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// IgnoreStructField can be used as TaggedStructReflector.UntaggedNameFunc
// to ignore fields that don't have TaggedStructReflector.NameTag.
func IgnoreStructField(string) string { return "" }

// TaggedStructReflector implements StructReflector
var _ StructReflector = new(TaggedStructReflector)

// TaggedStructReflector implements StructReflector with a struct field NameTag
// to be used for naming and a UntaggedNameFunc in case the NameTag is not set.
type TaggedStructReflector struct {
	// NameTag is the struct field tag to be used as column name
	NameTag string

	// Ignore will cause a struct field to be ignored if it has that name
	Ignore string

	PrimaryKey string
	ReadOnly   string
	Default    string

	// UntaggedNameFunc will be called with the struct field name to
	// return a column name in case the struct field has no tag named NameTag.
	// Use IgnoreStructField to skip untagged fields
	// or ToSnakeCase to derive column names from field names.
	UntaggedNameFunc func(fieldName string) string

	// FailOnUnmappedColumns controls whether ScanableStructFieldsForColumns returns an error
	// when query result columns have no mapped struct fields.
	// If false (the default), unmapped columns are silently ignored
	// by using discard scan destinations.
	FailOnUnmappedColumns bool

	// FailOnUnmappedStructFields controls whether ScanableStructFieldsForColumns returns an error
	// when the struct has mapped fields with no corresponding query result column.
	// If false (the default), unmapped struct fields are silently left unchanged.
	FailOnUnmappedStructFields bool

	// TypeWrappers are used to wrap struct field values as
	// driver.Valuer or sql.Scanner implementations
	// for custom serialization and deserialization of column values.
	TypeWrappers TypeWrappers
}

// NewTaggedStructReflector returns a TaggedStructReflector
// with the default "db" struct tag for column naming,
// "-" to ignore fields, and the flags "primarykey", "readonly", "default".
// Struct fields without a "db" tag are ignored (IgnoreStructField).
// Unmapped columns and struct fields do not cause errors.
// Optional typeWrappers are used for custom serialization/deserialization
// of struct field values as driver.Valuer or sql.Scanner implementations.
func NewTaggedStructReflector(typeWrappers ...TypeWrapper) *TaggedStructReflector {
	return &TaggedStructReflector{
		NameTag:                    "db",
		Ignore:                     "-",
		PrimaryKey:                 "primarykey",
		ReadOnly:                   "readonly",
		Default:                    "default",
		UntaggedNameFunc:           IgnoreStructField,
		FailOnUnmappedColumns:      false,
		FailOnUnmappedStructFields: false,
		TypeWrappers:               typeWrappers,
	}
}

// TableNameForStruct implements StructReflector.TableNameForStruct.
func (refl *TaggedStructReflector) TableNameForStruct(t reflect.Type) (table string, err error) {
	return TableNameForStruct(t, refl.NameTag)
}

// MapStructField implements StructReflector.MapStructField.
func (refl *TaggedStructReflector) MapStructField(field reflect.StructField) (column ColumnInfo, use bool) {
	if field.Anonymous {
		tag, hasTag := field.Tag.Lookup(refl.NameTag)
		if !hasTag {
			// Embedded struct fields are ok if not tagged with IgnoreName
			return ColumnInfo{}, true
		}
		columnName, _, _ := strings.Cut(tag, ",")
		columnName = strings.TrimSpace(columnName)
		// Embedded struct fields are ok if not tagged with IgnoreName
		return ColumnInfo{}, columnName != refl.Ignore
	}
	if !field.IsExported() {
		// Not exported struct fields that are not
		// anonymously embedded structs are not ok
		return ColumnInfo{}, false
	}

	if tag, hasTag := field.Tag.Lookup(refl.NameTag); hasTag {
		column.Name, tag, _ = strings.Cut(tag, ",")
		column.Name = strings.TrimSpace(column.Name)

		str, tag, ok := strings.Cut(tag, ",")
		for str != "" || ok {
			switch strings.TrimSpace(str) {
			case refl.PrimaryKey:
				column.PrimaryKey = true
			case refl.ReadOnly:
				column.ReadOnly = true
			case refl.Default:
				column.HasDefault = true
			}
			str, tag, ok = strings.Cut(tag, ",")
		}
	} else {
		column.Name = refl.UntaggedNameFunc(field.Name)
	}

	if column.Name == "" || column.Name == refl.Ignore {
		return ColumnInfo{}, false
	}
	return column, true
}

// ScanableStructFieldsForColumns implements StructReflector.ScanableStructFieldsForColumns.
func (refl *TaggedStructReflector) ScanableStructFieldsForColumns(structVal reflect.Value, columns []string) (scanables []any, err error) {
	if len(columns) == 0 {
		return nil, errors.New("no columns")
	}
	rs, err := reflectStruct(refl, structVal.Type())
	if err != nil {
		return nil, err
	}
	scanables = make([]any, len(columns))
	for i, col := range columns {
		idx, ok := rs.ColumnIndex[col]
		if !ok {
			continue
		}
		field := structVal.FieldByIndex(rs.Fields[idx].FieldIndex)
		if scanner := refl.TypeWrappers.WrapAsScanner(field); scanner != nil {
			scanables[i] = scanner
		} else {
			scanables[i] = field.Addr().Interface()
		}
	}
	for i, scanable := range scanables {
		if scanable != nil {
			continue // ok
		}
		if !refl.FailOnUnmappedColumns {
			// Use discard destination for unmapped columns
			scanables[i] = new(any)
			continue // ok with shim
		}
		// Fail on unmapped columns
		var nilCols strings.Builder
		for j, scanable := range scanables {
			if scanable != nil {
				continue
			}
			if nilCols.Len() > 0 {
				nilCols.WriteString(", ")
			}
			fmt.Fprintf(&nilCols, "column=%s, index=%d", columns[j], j)
		}
		return nil, fmt.Errorf("columns have no mapped struct fields in %s: %s", structVal.Type(), nilCols.String())
	}
	if refl.FailOnUnmappedStructFields {
		columnSet := make(map[string]struct{}, len(columns))
		for _, col := range columns {
			columnSet[col] = struct{}{}
		}
		var unmapped strings.Builder
		for _, f := range rs.Fields {
			if _, ok := columnSet[f.Column.Name]; !ok {
				if unmapped.Len() > 0 {
					unmapped.WriteString(", ")
				}
				unmapped.WriteString(f.Column.Name)
			}
		}
		if unmapped.Len() > 0 {
			return nil, fmt.Errorf("struct fields have no mapped columns in query result for %s: %s", structVal.Type(), unmapped.String())
		}
	}
	return scanables, nil
}

// PrimaryKeyColumnsOfStruct implements StructReflector.PrimaryKeyColumnsOfStruct.
func (refl *TaggedStructReflector) PrimaryKeyColumnsOfStruct(t reflect.Type) (columns []string, err error) {
	rs, err := reflectStruct(refl, t)
	if err != nil {
		return nil, err
	}
	for i := range rs.Fields {
		f := &rs.Fields[i]
		if f.Column.PrimaryKey {
			columns = append(columns, f.Column.Name)
		}
	}
	return columns, nil
}

// ReflectStructColumnsAndValues implements StructReflector.ReflectStructColumnsAndValues.
func (refl *TaggedStructReflector) ReflectStructColumnsAndValues(structVal reflect.Value, options ...QueryOption) (columns []ColumnInfo, values []any, err error) {
	rs, err := reflectStruct(refl, structVal.Type())
	if err != nil {
		return nil, nil, err
	}
	for i := range rs.Fields {
		f := &rs.Fields[i]
		if QueryOptionsIgnoreColumn(&f.Column, options) {
			continue
		}
		if QueryOptionsIgnoreStructField(&f.StructField, options) {
			continue
		}
		columns = append(columns, f.Column)
		field := structVal.FieldByIndex(f.FieldIndex)
		if valuer := refl.TypeWrappers.WrapAsValuer(field); valuer != nil {
			values = append(values, valuer)
		} else {
			values = append(values, field.Interface())
		}
	}
	return columns, values, nil
}

// ReflectStructColumnsFieldIndicesAndValues implements StructReflector.ReflectStructColumnsFieldIndicesAndValues.
func (refl *TaggedStructReflector) ReflectStructColumnsFieldIndicesAndValues(structVal reflect.Value, options ...QueryOption) (columns []ColumnInfo, indices [][]int, values []any, err error) {
	rs, err := reflectStruct(refl, structVal.Type())
	if err != nil {
		return nil, nil, nil, err
	}
	for i := range rs.Fields {
		f := &rs.Fields[i]
		if QueryOptionsIgnoreColumn(&f.Column, options) {
			continue
		}
		if QueryOptionsIgnoreStructField(&f.StructField, options) {
			continue
		}
		columns = append(columns, f.Column)
		indices = append(indices, f.FieldIndex)
		field := structVal.FieldByIndex(f.FieldIndex)
		if valuer := refl.TypeWrappers.WrapAsValuer(field); valuer != nil {
			values = append(values, valuer)
		} else {
			values = append(values, field.Interface())
		}
	}
	return columns, indices, values, nil
}

// ReflectStructValues implements StructReflector.ReflectStructValues.
func (refl *TaggedStructReflector) ReflectStructValues(structVal reflect.Value, options ...QueryOption) (values []any, err error) {
	rs, err := reflectStruct(refl, structVal.Type())
	if err != nil {
		return nil, err
	}
	for i := range rs.Fields {
		f := &rs.Fields[i]
		if QueryOptionsIgnoreColumn(&f.Column, options) {
			continue
		}
		if QueryOptionsIgnoreStructField(&f.StructField, options) {
			continue
		}
		field := structVal.FieldByIndex(f.FieldIndex)
		if valuer := refl.TypeWrappers.WrapAsValuer(field); valuer != nil {
			values = append(values, valuer)
		} else {
			values = append(values, field.Interface())
		}
	}
	return values, nil
}

// ReflectStructColumns implements StructReflector.ReflectStructColumns.
func (refl *TaggedStructReflector) ReflectStructColumns(structType reflect.Type, options ...QueryOption) (columns []ColumnInfo, err error) {
	rs, err := reflectStruct(refl, structType)
	if err != nil {
		return nil, err
	}
	for i := range rs.Fields {
		f := &rs.Fields[i]
		if QueryOptionsIgnoreColumn(&f.Column, options) {
			continue
		}
		if QueryOptionsIgnoreStructField(&f.StructField, options) {
			continue
		}
		columns = append(columns, f.Column)
	}
	return columns, nil
}

// ReflectStructColumnsAndFields implements StructReflector.ReflectStructColumnsAndFields.
func (refl *TaggedStructReflector) ReflectStructColumnsAndFields(structVal reflect.Value, options ...QueryOption) (columns []ColumnInfo, fields []reflect.Type, err error) {
	rs, err := reflectStruct(refl, structVal.Type())
	if err != nil {
		return nil, nil, err
	}
	for i := range rs.Fields {
		f := &rs.Fields[i]
		if QueryOptionsIgnoreColumn(&f.Column, options) {
			continue
		}
		if QueryOptionsIgnoreStructField(&f.StructField, options) {
			continue
		}
		columns = append(columns, f.Column)
		fields = append(fields, f.StructField.Type)
	}
	return columns, fields, nil
}

func (refl *TaggedStructReflector) String() string {
	return fmt.Sprintf("NameTag: %q", refl.NameTag)
}
