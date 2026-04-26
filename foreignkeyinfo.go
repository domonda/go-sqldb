package sqldb

// ForeignKeyInfo describes a single foreign key constraint declared on
// a table. It is the return shape of
// [Information.ForeignKeys].
//
// Composite foreign keys (FKs spanning more than one column) are
// represented by a single ForeignKeyInfo whose Columns and
// ReferencedColumns slices have matching length and ordering: the i-th
// local column references the i-th remote column. Ordering follows the
// constraint declaration, not column declaration order on either
// table.
type ForeignKeyInfo struct {
	// Name is the constraint name as recorded in the catalog. It is
	// unique within the owning table and is the identifier to use with
	// `ALTER TABLE ... DROP CONSTRAINT <name>`.
	Name string

	// Columns lists the local columns that participate in the
	// constraint, in constraint-declaration order. For a single-column
	// FK this has length 1; for a composite FK, length > 1.
	Columns []string

	// ReferencedTable is the table the constraint points at,
	// schema-qualified as `schema.name`.
	ReferencedTable string

	// ReferencedColumns lists the remote columns referenced by this
	// constraint, in matching order with Columns. For a single-column
	// FK this has length 1; for a composite FK, length matches len(Columns).
	ReferencedColumns []string

	// OnDelete is the referential action triggered when a referenced
	// row is deleted. Standard values per ISO/IEC 9075-2:
	//
	//   - "NO ACTION"   — default; the database checks at the end of
	//                     the statement. (PostgreSQL: deferred check.)
	//   - "RESTRICT"    — like NO ACTION but checked immediately.
	//   - "CASCADE"     — delete the dependent rows too.
	//   - "SET NULL"    — null out the local FK columns.
	//   - "SET DEFAULT" — reset the local FK columns to their default.
	//
	// Empty string when the catalog does not report an action (rare;
	// usually means NO ACTION).
	OnDelete string

	// OnUpdate is the referential action triggered when a referenced
	// row's key is updated. Same value space as OnDelete.
	//
	// Many engines do not enforce ON UPDATE actions reliably (notably
	// MySQL/InnoDB does, SQL Server does, PostgreSQL does, Oracle does
	// not support cascading updates at all and the catalog will report
	// "NO ACTION").
	OnUpdate string
}
