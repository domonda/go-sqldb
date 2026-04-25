package information

import (
	"context"
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

// Column maps a row from information_schema.columns.
//
// Vendor support:
//   - PostgreSQL: all fields populated.
//   - MySQL/MariaDB: TableCatalog is always "def"; CharacterSet*,
//     Collation*, Domain*, UDT*, Scope*, MaximumCardinality, DTDIdentifier,
//     IsSelfReferencing are not populated. Identity* is populated on
//     MySQL 8.0+; IsGenerated/GenerationExpression on MySQL 5.7+.
//   - SQL Server: only the ISO base columns are populated; identity,
//     generation, scope, UDT, domain, character set and collation
//     metadata require sys.* catalog views instead.
//   - SQLite, Oracle: information_schema is not implemented.
type Column struct {
	TableCatalog           String `db:"table_catalog"`
	TableSchema            String `db:"table_schema"`
	TableName              String `db:"table_name"`
	ColumnName             String `db:"column_name"`
	OrdinalPosition        int    `db:"ordinal_position"`
	ColumnDefault          String `db:"column_default"`
	IsNullable             YesNo  `db:"is_nullable"`
	DataType               String `db:"data_type"`
	CharacterMaximumLength *int   `db:"character_maximum_length"`
	CharacterOctetLength   *int   `db:"character_octet_length"`
	NumericPrecision       *int   `db:"numeric_precision"`
	NumericPrecisionRadix  *int   `db:"numeric_precision_radix"`
	NumericScale           *int   `db:"numeric_scale"`
	DatetimePrecision      *int   `db:"datetime_precision"`
	IntervalType           String `db:"interval_type"`
	IntervalPrecision      *int   `db:"interval_precision"`
	CharacterSetCatalog    String `db:"character_set_catalog"`
	CharacterSetSchema     String `db:"character_set_schema"`
	CharacterSetName       String `db:"character_set_name"`
	CollationCatalog       String `db:"collation_catalog"`
	CollationSchema        String `db:"collation_schema"`
	CollationName          String `db:"collation_name"`
	DomainCatalog          String `db:"domain_catalog"`
	DomainSchema           String `db:"domain_schema"`
	DomainName             String `db:"domain_name"`
	UDTCatalog             String `db:"udt_catalog"`
	UDTSchema              String `db:"udt_schema"`
	UDTName                String `db:"udt_name"`
	ScopeCatalog           String `db:"scope_catalog"`
	ScopeSchema            String `db:"scope_schema"`
	ScopeName              String `db:"scope_name"`
	MaximumCardinality     *int   `db:"maximum_cardinality"`
	DTDIdentifier          String `db:"dtd_identifier"`
	IsSelfReferencing      YesNo  `db:"is_self_referencing"`
	IsIdentity             YesNo  `db:"is_identity"`
	IdentityGeneration     String `db:"identity_generation"`
	IdentityStart          String `db:"identity_start"`
	IdentityIncrement      String `db:"identity_increment"`
	IdentityMaximum        String `db:"identity_maximum"`
	IdentityMinimum        String `db:"identity_minimum"`
	IdentityCycle          YesNo  `db:"identity_cycle"`
	IsGenerated            String `db:"is_generated"`
	GenerationExpression   String `db:"generation_expression"`
	IsUpdatable            YesNo  `db:"is_updatable"`
}

// KeyColumnUsage maps a row from information_schema.key_column_usage.
//
// Vendor support: PostgreSQL, MySQL, MariaDB, SQL Server. SQLite and
// Oracle do not expose information_schema.
type KeyColumnUsage struct {
	ConstraintCatalog          String `db:"constraint_catalog"`
	ConstraintSchema           String `db:"constraint_schema"`
	ConstraintName             String `db:"constraint_name"`
	TableCatalog               String `db:"table_catalog"`
	TableSchema                String `db:"table_schema"`
	TableName                  String `db:"table_name"`
	ColumnName                 String `db:"column_name"`
	OrdinalPosition            int    `db:"ordinal_position"`
	PositionInUniqueConstraint *int   `db:"position_in_unique_constraint"`
}

// ColumnExists reports whether a column exists in the given table.
//
// table may be schema-qualified as "schema.table" or unqualified. When
// unqualified the schema is not constrained, so the function returns
// true if any schema in the current database contains a table with the
// given name and a column with the given name.
//
// Vendor notes:
//   - PostgreSQL, MySQL, MariaDB, SQL Server: supported.
//   - SQLite, Oracle: not supported (no information_schema).
func ColumnExists(ctx context.Context, conn sqldb.Connection, table, column string) (bool, error) {
	var (
		query string
		args  []any
	)
	if schema, name, ok := strings.Cut(table, "."); ok {
		query = fmt.Sprintf(
			/*sql*/ `SELECT CASE WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = %s AND table_name = %s AND column_name = %s) THEN 1 ELSE 0 END`,
			conn.FormatPlaceholder(0),
			conn.FormatPlaceholder(1),
			conn.FormatPlaceholder(2),
		)
		args = []any{schema, name, column}
	} else {
		query = fmt.Sprintf(
			/*sql*/ `SELECT CASE WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = %s AND column_name = %s) THEN 1 ELSE 0 END`,
			conn.FormatPlaceholder(0),
			conn.FormatPlaceholder(1),
		)
		args = []any{table, column}
	}
	n, err := sqldb.QueryRowAs[int](ctx, conn, structReflector, conn, query, args...)
	return n != 0, err
}
