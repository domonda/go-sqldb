package sqldb

// ColumnInfo holds metadata about a database column.
//
// A ColumnInfo can come from either of two sources, and the meaning
// of every field depends on which source produced it:
//
//   - Struct reflection (e.g. [StructReflector.MapStructField],
//     [StructReflector.ReflectStructColumns],
//     [StructReflector.ReflectStructColumnsAndValues]). The metadata
//     describes the Go side of the mapping: Name comes from the
//     `db:"..."` tag (or the field-name fallback), Type is the Go
//     type of the mapped struct field as returned by
//     [reflect.StructField.Type.String] (e.g. "string", "int",
//     "*time.Time", "uu.ID"), and the boolean flags reflect tag
//     options (`primarykey`, `default`, `readonly`). Generated is
//     always false on this path — the struct-tag vocabulary has no
//     equivalent.
//
//   - Database introspection via the [Information] interface
//     (`Information.Columns`). The metadata describes the catalog
//     side: Name is the catalog column name, Type is the vendor's
//     catalog type spelling (e.g. PostgreSQL "integer" / "character
//     varying" / "uuid", SQL Server "int" / "uniqueidentifier",
//     Oracle "NUMBER" / "VARCHAR2"), and the boolean flags are
//     populated from vendor catalog views — see each field's godoc
//     for the exact predicate.
//
// A third lightweight source is the [Values] map, where ColumnInfo
// values carry only Name (the map key); Type and the boolean flags
// are zero because the map carries no schema information.
type ColumnInfo struct {
	// Name is the column name.
	//
	// From struct reflection: the value of the `db:"..."` tag, with
	// any tag options stripped. If the field has no tag, the struct
	// field name itself is used (or transformed by the reflector's
	// `UntaggedNameFunc`, e.g. `ToSnakeCase`).
	//
	// From database introspection: the catalog column name, as
	// stored. On vendors that case-fold unquoted identifiers (Oracle)
	// the catalog returns the uppercase form.
	//
	// From a [Values] map: the map key.
	//
	// Empty Name has a special meaning when the ColumnInfo was
	// produced by [StructReflector.MapStructField]: the entry
	// represents an embedded struct field whose own fields should be
	// flattened into the parent's column set. It is NOT a real
	// column, and the other fields (Type, PrimaryKey, HasDefault,
	// ReadOnly, Generated) are not meaningful in that case. Callers
	// iterating reflection results must detect Name == "" and recurse
	// into the field's Type instead of treating the entry as a column.
	// ColumnInfo values from database introspection or [Values] maps
	// always have a non-empty Name.
	Name string

	// Type is a free-form description of the column's type, with the
	// exact spelling depending on the source.
	//
	// From struct reflection: the Go type as
	// [reflect.StructField.Type.String] returns it (e.g. `string`,
	// `int`, `*time.Time`, `uu.ID`, `sql.NullString`).
	//
	// From database introspection: the catalog's type string. This
	// is the format produced by the vendor's catalog, including any
	// length / precision modifiers (e.g. PostgreSQL `character
	// varying(255)`, SQL Server `nvarchar`, Oracle `NUMBER`,
	// MySQL `int unsigned`, SQLite `INTEGER`). It is intentionally
	// vendor-specific — translating it to a portable Go type is the
	// caller's job.
	//
	// Type may be empty when the source did not record type
	// information (e.g. a ColumnInfo built from a [Values] map, where
	// only the column name is known up front).
	Type string

	// PrimaryKey is true when the column participates in the
	// table's primary key constraint.
	//
	// From struct reflection: the field has the `primarykey` tag
	// option (e.g. `db:"id,primarykey"`). Used by struct-based
	// update / upsert / query-by-PK to build WHERE clauses.
	//
	// From database introspection: the catalog reports the column as
	// part of the table's primary key. PrimaryKey is set independent
	// of constraint-declaration order; use [Information.PrimaryKey]
	// when the order matters (e.g. for composite keys whose index
	// order differs from column-declaration order).
	PrimaryKey bool

	// HasDefault is true when omitting the column from an INSERT is
	// safe because the database will supply a value.
	//
	// From struct reflection: the field has the `default` tag option
	// (e.g. `db:"id,primarykey,default"`). Combined with
	// [IgnoreHasDefault], this lets struct inserts skip zero-valued
	// fields whose database column has a default.
	//
	// From database introspection: the catalog reports any of —
	//   - a literal `DEFAULT` clause (`column_default IS NOT NULL`),
	//   - an identity column / `SERIAL` / `AUTO_INCREMENT`,
	//   - on PostgreSQL and Oracle, an identity column regardless of
	//     `ALWAYS` vs `BY DEFAULT`,
	//   - on some vendors, a generated column whose value the engine
	//     fills in.
	HasDefault bool

	// ReadOnly is true when the column should be excluded from
	// generated INSERT and UPDATE statements. It is the broader
	// "do not write to this column" hint and is the field consumed
	// by [IgnoreReadOnly] and the `db:"col,readonly"` struct tag.
	//
	// When populated by database introspection, ReadOnly is the
	// union of multiple catalog conditions:
	//
	//   - The column is GENERATED (see [ColumnInfo.Generated]).
	//   - The column is `GENERATED ALWAYS AS IDENTITY` (PostgreSQL,
	//     Oracle): a sequence supplies the value and user-supplied
	//     values are rejected.
	//
	// When populated by struct reflection, ReadOnly mirrors the
	// `readonly` tag option, which the developer may set for any
	// reason — including columns kept up to date by triggers,
	// audit fields the application never writes to, or
	// trigger-managed search vectors.
	//
	// Use Generated when you need the catalog-precise "expression
	// or stored-generated column" predicate; use ReadOnly when you
	// need "do not include in INSERT/UPDATE."
	ReadOnly bool

	// Generated is true when the column is a SQL-standard
	// `GENERATED ALWAYS AS (expression) [STORED|VIRTUAL]` column —
	// the database derives the value from other columns or a
	// constant expression and the user cannot write to it
	// directly. Vendor catalog mapping:
	//
	//   - PostgreSQL: `pg_attribute.attgenerated <> ''` (matches
	//     stored 's' and, on PostgreSQL 18+, virtual 'v'; '' is the
	//     empty/no-generation case).
	//   - MySQL/MariaDB: `extra` column matches `GENERATED`,
	//     `STORED`, or `VIRTUAL`.
	//   - SQL Server: `sys.columns.is_computed = 1`.
	//   - Oracle: `all_tab_cols.virtual_column = 'YES'`.
	//   - SQLite: `PRAGMA table_xinfo.hidden` is `2` (stored) or
	//     `3` (virtual).
	//
	// Generated is strictly narrower than [ColumnInfo.ReadOnly]:
	// every Generated column is also ReadOnly, but the reverse is
	// not true. Identity-always columns on PostgreSQL and Oracle
	// are ReadOnly but NOT Generated. Columns marked
	// `db:"col,readonly"` by the developer are ReadOnly but
	// usually not Generated.
	//
	// When populated by struct reflection, Generated is always
	// false because the struct-tag layer has no notion of
	// catalog-derived generation.
	Generated bool
}
