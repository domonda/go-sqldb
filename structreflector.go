package sqldb

import (
	"reflect"
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

	// ScanableStructFieldsForColumns returns a slice of values
	// corresponding to the given query result column names,
	// where each value is either a pointer to a scanable type
	// or an implementation of the sql.Scanner interface,
	// suitable for use with Rows.Scan.
	ScanableStructFieldsForColumns(structVal reflect.Value, columns []string) (scanable []any, err error)

	// PrimaryKeyColumnsOfStruct returns the column names of the primary key fields
	// for the given struct type.
	PrimaryKeyColumnsOfStruct(t reflect.Type) (columns []string, err error)

	// ReflectStructColumnsAndValues returns the column metadata and corresponding field values
	// for the given struct value, filtered by the provided query options.
	// Field values are wrapped using WrapAsValuer if a TypeWrapper
	// handles the field's type, otherwise the plain field value is used.
	ReflectStructColumnsAndValues(structVal reflect.Value, options ...QueryOption) (columns []ColumnInfo, values []any, err error)

	// ReflectStructColumnsFieldIndicesAndValues returns the column metadata, struct field indices,
	// and corresponding field values for the given struct value, filtered by the provided query options.
	// Field values are wrapped using WrapAsValuer if a TypeWrapper
	// handles the field's type, otherwise the plain field value is used.
	ReflectStructColumnsFieldIndicesAndValues(structVal reflect.Value, options ...QueryOption) (columns []ColumnInfo, indices [][]int, values []any, err error)

	// ReflectStructValues returns the field values of the given struct value
	// for mapped columns, filtered by the provided query options.
	// Field values are wrapped using WrapAsValuer if a TypeWrapper
	// handles the field's type, otherwise the plain field value is used.
	ReflectStructValues(structVal reflect.Value, options ...QueryOption) (values []any, err error)

	// ReflectStructColumns returns the column metadata for the given struct type,
	// filtered by the provided query options.
	ReflectStructColumns(structType reflect.Type, options ...QueryOption) (columns []ColumnInfo, err error)

	// ReflectStructColumnsAndFields returns the column metadata and corresponding field types
	// for the given struct value, filtered by the provided query options.
	ReflectStructColumnsAndFields(structVal reflect.Value, options ...QueryOption) (columns []ColumnInfo, fields []reflect.Type, err error)
}
