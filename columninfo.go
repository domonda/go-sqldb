package sqldb

// ColumnInfo holds metadata about a database column.
//
// A ColumnInfo can come from either of two sources:
//
//   - Struct reflection (e.g. [StructReflector.MapStructField],
//     [StructReflector.ReflectStructColumns],
//     [StructReflector.ReflectStructColumnsAndValues]). In this case
//     the metadata describes the Go side of the mapping: Name comes
//     from the `db:"..."` tag (or the field name fallback), and Type
//     is the Go type of the mapped struct field as returned by
//     [reflect.StructField.Type.String] (e.g. "string", "int",
//     "*time.Time", "uu.ID").
//
//   - Database introspection. In this case the metadata describes
//     the database side: Name is the catalog column name and Type is
//     whatever string the catalog reports for the column type
//     (e.g. "integer", "character varying", "uuid",
//     "uniqueidentifier"). The package itself does not currently
//     populate ColumnInfo from the catalog; this convention is
//     reserved for callers that build their own ColumnInfo values
//     from queries against information_schema or vendor sys.* views.
//
// Type may be empty when the source did not record type information
// (for example, ColumnInfo built from a [Values] map, where only the
// column name is known up front).
type ColumnInfo struct {
	// Name is the column name in the database.
	//
	// An empty Name has a special meaning when the ColumnInfo was
	// produced by [StructReflector.MapStructField]: the value
	// represents an embedded struct field whose own fields should be
	// flattened into the parent's column set. It is NOT a real column
	// and the other fields (Type, PrimaryKey, HasDefault, ReadOnly)
	// are not meaningful in that case. Callers iterating reflection
	// results should detect Name == "" and recurse into the field's
	// Type instead of treating the entry as a column.
	//
	// For ColumnInfo values that originate from database introspection
	// or from a [Values] map, Name is always a real column name and
	// should never be empty.
	Name string

	// Type is a free-form description of the column's type. Its
	// meaning depends on how the ColumnInfo was constructed; see
	// the [ColumnInfo] doc comment.
	Type string

	PrimaryKey bool
	HasDefault bool
	ReadOnly   bool
}
