package sqldb

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/domonda/go-types/strutil"
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
	// An empty string for column and true for use indicates an
	// embedded struct field whose fields should be mapped recursively.
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

// DefaultStructFieldMapping provides the default StructFieldTagNaming
// using "db" as NameTag and IgnoreStructField as UntaggedNameFunc.
// Implements StructFieldMapper.
var DefaultStructFieldMapping = NewTaggedStructFieldMapping()

// TaggedStructFieldMapping implements StructFieldMapper
var _ StructFieldMapper = new(TaggedStructFieldMapping)

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
// Whitespace, symbol, and punctuation characters
// will be replace by '_'.
func ToSnakeCase(s string) string {
	return strutil.ToSnakeCase(s)
}

type MappedStructField struct {
	Field  reflect.StructField
	Table  string
	Column string
	Flags  FieldFlag
}

type MappedStruct struct {
	Type         reflect.Type
	Table        string
	Fields       []MappedStructField
	ColumnFields map[string]*MappedStructField
}

var (
	mappedStructTypeCache    = make(map[StructFieldMapper]map[reflect.Type]*MappedStruct)
	mappedStructTypeCacheMtx sync.Mutex
)

func mapStructType(mapper StructFieldMapper, structType reflect.Type, mapped *MappedStruct) error {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		table, column, flags, use := mapper.MapStructField(field)
		if !use {
			continue
		}

		if table != "" {
			if mapped.Table != "" && table != mapped.Table {
				return fmt.Errorf("conflicting tables %s and %s found in struct %s", mapped.Table, table, structType)
			}
			mapped.Table = table
		}

		if column == "" {
			// Embedded struct field
			t := field.Type
			if t.Kind() == reflect.Pointer {
				t = t.Elem()
			}
			err := mapStructType(mapper, t, mapped)
			if err != nil {
				return err
			}
			continue
		}

		if _, exists := mapped.ColumnFields[column]; exists {
			return fmt.Errorf("duplicate mapped column %s onto field %s of struct %s", column, field.Name, structType)
		}

		mapped.Fields = append(mapped.Fields, MappedStructField{
			Field:  field,
			Table:  table,
			Column: column,
			Flags:  flags,
		})
		mapped.ColumnFields[column] = &mapped.Fields[len(mapped.Fields)-1]
	}
	return nil
}

func MapStructType(mapper StructFieldMapper, structType reflect.Type) (*MappedStruct, error) {
	mappedStructTypeCacheMtx.Lock()
	defer mappedStructTypeCacheMtx.Unlock()

	mapped := mappedStructTypeCache[mapper][structType]
	if mapped != nil {
		return mapped, nil
	}

	mapped = &MappedStruct{Type: structType}
	err := mapStructType(mapper, structType, mapped)
	if err != nil {
		return nil, err
	}
	if mappedStructTypeCache[mapper] == nil {
		mappedStructTypeCache[mapper] = make(map[reflect.Type]*MappedStruct)
	}
	mappedStructTypeCache[mapper][structType] = mapped
	return mapped, nil
}

func MapStruct(mapper StructFieldMapper, s any) (*MappedStruct, error) {
	v := reflect.ValueOf(s)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct but got %T", s)
	}
	if !v.CanAddr() {
		return nil, errors.New("struct can't be addressed")
	}

	return MapStructType(mapper, v.Type())
}

// func MapStructFieldPointers(mapper StructFieldMapper, strct any) (colFieldPtrs map[string]any, table string, err error) {
// 	v := reflect.ValueOf(strct)
// 	for v.Kind() == reflect.Ptr && !v.IsNil() {
// 		v = v.Elem()
// 	}
// 	if v.Kind() != reflect.Struct {
// 		return nil, "", fmt.Errorf("expected struct but got %T", strct)
// 	}
// 	if !v.CanAddr() {
// 		return nil, "", errors.New("struct can't be addressed")
// 	}

// 	mapped, err := getOrCreateMappedStruct(mapper, v.Type())
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	colFieldPtrs = make(map[string]any, len(mapped.Fields))
// 	for column, mapped := range mapped.ColumnFields {
// 		field, err := v.FieldByIndexErr(mapped.Field.Index)
// 		if err != nil {
// 			return nil, "", err
// 		}
// 		colFieldPtrs[column] = field.Addr().Interface()
// 	}
// 	return colFieldPtrs, mapped.Table, nil
// }

// func MapStructFieldColumnPointers(mapper StructFieldMapper, structVal any, columns []string) (ptrs []any, table string, colsWithoutField, fieldsWithoutCol []string, err error) {
// 	v := reflect.ValueOf(structVal)
// 	for v.Kind() == reflect.Ptr && !v.IsNil() {
// 		v = v.Elem()
// 	}
// 	if v.Kind() != reflect.Struct {
// 		return nil, "", fmt.Errorf("expected struct but got %T", structVal)
// 	}
// 	if !v.CanAddr() {
// 		return nil, "", errors.New("struct can't be addressed")
// 	}

// 	mapped, err := MapStructType(mapper, v.Type())
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	colFieldPtrs = make(map[string]any, len(mapped.Fields))
// 	for column, mapped := range mapped.ColumnFields {
// 		field, err := v.FieldByIndexErr(mapped.Field.Index)
// 		if err != nil {
// 			return nil, "", err
// 		}
// 		colFieldPtrs[column] = field.Addr().Interface()
// 	}
// 	return colFieldPtrs, mapped.Table, nil
// }
