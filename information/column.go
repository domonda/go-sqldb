package information

import (
	"context"
	"strings"

	"github.com/domonda/go-errs"
	"github.com/domonda/go-sqldb/db"
)

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

func ColumnExists(ctx context.Context, table, column string) (exists bool, err error) {
	defer errs.WrapWithFuncParams(&err, ctx, table, column)

	tableSchema, tableName, ok := strings.Cut(table, ".")
	if !ok {
		tableSchema = "public"
		tableName = table
	}

	err = db.QueryRow(ctx,
		`select exists(
			select from information_schema.columns
			where table_schema = $1
				and table_name = $2
				and column_name = $3
		)`,
		tableSchema,
		tableName,
		column,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
