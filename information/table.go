package information

import (
	"context"
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

// Table maps a row from information_schema.tables.
//
// Vendor support:
//   - PostgreSQL: all fields populated.
//   - MySQL/MariaDB: TableCatalog is always the literal string "def".
//     The view does not contain SelfReferencingColumnName,
//     ReferenceGeneration, UserDefinedType*, IsInsertableInto, IsTyped,
//     or CommitAction at all (MariaDB/MySQL use a different set of
//     extension columns like ENGINE / ROW_FORMAT that are not exposed
//     by this struct), so those fields scan as their zero values.
//   - SQL Server: only the ISO base columns (TableCatalog, TableSchema,
//     TableName, TableType) are populated; the remaining ISO extension
//     fields scan as empty.
//   - SQLite, Oracle: information_schema is not implemented; queries that
//     reference it (including the helpers in this package) will fail.
type Table struct {
	TableCatalog              String `db:"table_catalog"`
	TableSchema               String `db:"table_schema"`
	TableName                 String `db:"table_name"`
	TableType                 String `db:"table_type"`
	SelfReferencingColumnName String `db:"self_referencing_column_name"`
	ReferenceGeneration       String `db:"reference_generation"`
	UserDefinedTypeCatalog    String `db:"user_defined_type_catalog"`
	UserDefinedTypeSchema     String `db:"user_defined_type_schema"`
	UserDefinedTypeName       String `db:"user_defined_type_name"`
	IsInsertableInto          YesNo  `db:"is_insertable_into"`
	IsTyped                   YesNo  `db:"is_typed"`
	CommitAction              String `db:"commit_action"`
}

// GetTable returns the information_schema.tables row for the given catalog,
// schema, and table name.
//
// Vendor notes:
//   - PostgreSQL: catalog is the database name.
//   - MySQL/MariaDB: catalog must be the literal string "def".
//   - SQL Server: catalog is the database name; many returned fields will
//     be empty (see [Table]).
//   - SQLite, Oracle: not supported (no information_schema).
func GetTable(ctx context.Context, conn sqldb.Connection, catalog, schema, name string) (*Table, error) {
	query := fmt.Sprintf(
		/*sql*/ `SELECT * FROM information_schema.tables WHERE table_catalog = %s AND table_schema = %s AND table_name = %s`,
		conn.FormatPlaceholder(0),
		conn.FormatPlaceholder(1),
		conn.FormatPlaceholder(2),
	)
	return sqldb.QueryRowAs[*Table](ctx, conn, structReflector, conn, query, catalog, schema, name)
}

// TableExists reports whether a table exists in information_schema.tables.
//
// qualifiedName may be "schema.table" or just "table". When unqualified
// the schema is not constrained, so the function returns true if any
// schema in the current database contains a table with that name.
//
// Vendor notes:
//   - PostgreSQL, MySQL, MariaDB, SQL Server: supported.
//   - SQLite, Oracle: not supported (no information_schema).
func TableExists(ctx context.Context, conn sqldb.Connection, qualifiedName string) (bool, error) {
	var (
		query string
		args  []any
	)
	if schema, name, ok := strings.Cut(qualifiedName, "."); ok {
		query = fmt.Sprintf(
			/*sql*/ `SELECT CASE WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = %s AND table_name = %s) THEN 1 ELSE 0 END`,
			conn.FormatPlaceholder(0),
			conn.FormatPlaceholder(1),
		)
		args = []any{schema, name}
	} else {
		query = fmt.Sprintf(
			/*sql*/ `SELECT CASE WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = %s) THEN 1 ELSE 0 END`,
			conn.FormatPlaceholder(0),
		)
		args = []any{qualifiedName}
	}
	n, err := sqldb.QueryRowAs[int](ctx, conn, structReflector, conn, query, args...)
	return n != 0, err
}

// GetAllTables returns all rows from information_schema.tables.
//
// Vendor notes: see [Table] for which fields are populated on each
// vendor. SQLite and Oracle do not expose information_schema.
func GetAllTables(ctx context.Context, conn sqldb.Connection) ([]*Table, error) {
	return sqldb.QueryRowsAsSlice[*Table](ctx, conn, structReflector, conn, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `SELECT * FROM information_schema.tables`,
	)
}
