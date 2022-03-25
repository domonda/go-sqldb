package information

import (
	"fmt"
)

// YesNo is a bool type that implements the sql.Scanner
// interface for the information_schema.yes_or_no type.
type YesNo bool

// func (y YesNo) String() string {
// 	if y {
// 		return "YES"
// 	} else {
// 		return "NO"
// 	}
// }

func (y *YesNo) Scan(value any) error {
	switch x := value.(type) {
	case bool:
		*y = YesNo(x)

	case string:
		switch x {
		case "YES":
			*y = true
		case "NO":
			*y = true
		default:
			return fmt.Errorf("can't scan SQL value %q as YesNo", value)
		}

	default:
		return fmt.Errorf("can't scan SQL value of type %T as YesNo", value)
	}
	return nil
}

// String is a string that implements the sql.Scanner
// interface to scan NULL as an empty string.
type String string

func (y *String) Scan(value any) error {
	switch x := value.(type) {
	case nil:
		*y = ""

	case string:
		*y = String(x)

	case []byte:
		*y = String(x)

	default:
		return fmt.Errorf("can't scan SQL value of type %T as String", value)
	}
	return nil
}

type Schema struct {
	CatalogName                String `db:"catalog_name"`
	SchemaName                 String `db:"schema_name"`
	SchemaOwner                String `db:"schema_owner"`
	DefaultCharacterSetCatalog String `db:"default_character_set_catalog"`
	DefaultCharacterSetSchema  String `db:"default_character_set_schema"`
	DefaultCharacterSetName    String `db:"default_character_set_name"`
	SqlPath                    String `db:"sql_path"`
}

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

type View struct {
	CatalogName                String `db:"catalog_name"`
	SchemaName                 String `db:"schema_name"`
	SchemaOwner                String `db:"schema_owner"`
	DefaultCharacterSetCatalog String `db:"default_character_set_catalog"`
	DefaultCharacterSetSchema  String `db:"default_character_set_schema"`
	DefaultCharacterSetName    String `db:"default_character_set_name"`
	SqlPath                    String `db:"sql_path"`
}

type Column struct {
	TableCatalog           String `db:"table_catalog"`
	TableSchema            String `db:"table_schema"`
	TableName              String `db:"table_name"`
	ColumnName             String `db:"column_name"`
	OrdinalPosition        int    `db:"ordinal_position"`
	ColumnDefault          String `db:"column_default"`
	IsNullable             YesNo  `db:"is_nullable"`
	DataType               String `db:"data_type"`
	CharacterMaximumLength int    `db:"character_maximum_length"`
	CharacterOctetLength   int    `db:"character_octet_length"`
	NumericPrecision       int    `db:"numeric_precision"`
	NumericPrecisionRadix  int    `db:"numeric_precision_radix"`
	NumericScale           int    `db:"numeric_scale"`
	DatetimePrecision      int    `db:"datetime_precision"`
	IntervalType           String `db:"interval_type"`
	IntervalPrecision      int    `db:"interval_precision"`
	CharacterSetCatalog    String `db:"character_set_catalog"`
	CharacterSetSchema     String `db:"character_set_schema"`
	CharacterSetName       String `db:"character_set_name"`
	CollationCatalog       String `db:"collation_catalog"`
	CollationSchema        String `db:"collation_schema"`
	CollationName          String `db:"collation_name"`
	DomainCatalog          String `db:"domain_catalog"`
	DomainSchema           String `db:"domain_schema"`
	DomainName             String `db:"domain_name"`
	UdtCatalog             String `db:"udt_catalog"`
	UdtSchema              String `db:"udt_schema"`
	UdtName                String `db:"udt_name"`
	ScopeCatalog           String `db:"scope_catalog"`
	ScopeSchema            String `db:"scope_schema"`
	ScopeName              String `db:"scope_name"`
	MaximumCardinality     int    `db:"maximum_cardinality"`
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
	PositionInUniqueConstraint int    `db:"position_in_unique_constraint"`
}

type Domains struct {
	DomainCatalog          String `db:"domain_catalog"`
	DomainSchema           String `db:"domain_schema"`
	DomainName             String `db:"domain_name"`
	DataType               String `db:"data_type"`
	CharacterMaximumLength int    `db:"character_maximum_length"`
	CharacterOctetLength   int    `db:"character_octet_length"`
	CharacterSetCatalog    String `db:"character_set_catalog"`
	CharacterSetSchema     String `db:"character_set_schema"`
	CharacterSetName       String `db:"character_set_name"`
	CollationCatalog       String `db:"collation_catalog"`
	CollationSchema        String `db:"collation_schema"`
	CollationName          String `db:"collation_name"`
	NumericPrecision       int    `db:"numeric_precision"`
	NumericPrecisionRadix  int    `db:"numeric_precision_radix"`
	NumericScale           int    `db:"numeric_scale"`
	DatetimePrecision      int    `db:"datetime_precision"`
	IntervalType           String `db:"interval_type"`
	IntervalPrecision      int    `db:"interval_precision"`
	DomainDefault          String `db:"domain_default"`
	UdtCatalog             String `db:"udt_catalog"`
	UdtSchema              String `db:"udt_schema"`
	UdtName                String `db:"udt_name"`
	ScopeCatalog           String `db:"scope_catalog"`
	ScopeSchema            String `db:"scope_schema"`
	ScopeName              String `db:"scope_name"`
	MaximumCardinality     int    `db:"maximum_cardinality"`
	DTDIdentifier          String `db:"dtd_identifier"`
}

type CheckConstraints struct {
	ConstraintCatalog String `db:"constraint_catalog"`
	ConstraintSchema  String `db:"constraint_schema"`
	ConstraintName    String `db:"constraint_name"`
	CheckClause       String `db:"check_clause"`
}
