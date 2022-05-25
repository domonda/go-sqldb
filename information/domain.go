package information

type Domain struct {
	DomainCatalog          String `db:"domain_catalog"`
	DomainSchema           String `db:"domain_schema"`
	DomainName             String `db:"domain_name"`
	DataType               String `db:"data_type"`
	CharacterMaximumLength *int   `db:"character_maximum_length"`
	CharacterOctetLength   *int   `db:"character_octet_length"`
	CharacterSetCatalog    String `db:"character_set_catalog"`
	CharacterSetSchema     String `db:"character_set_schema"`
	CharacterSetName       String `db:"character_set_name"`
	CollationCatalog       String `db:"collation_catalog"`
	CollationSchema        String `db:"collation_schema"`
	CollationName          String `db:"collation_name"`
	NumericPrecision       *int   `db:"numeric_precision"`
	NumericPrecisionRadix  *int   `db:"numeric_precision_radix"`
	NumericScale           *int   `db:"numeric_scale"`
	DatetimePrecision      *int   `db:"datetime_precision"`
	IntervalType           String `db:"interval_type"`
	IntervalPrecision      *int   `db:"interval_precision"`
	DomainDefault          String `db:"domain_default"`
	UDTCatalog             String `db:"udt_catalog"`
	UDTSchema              String `db:"udt_schema"`
	UDTName                String `db:"udt_name"`
	ScopeCatalog           String `db:"scope_catalog"`
	ScopeSchema            String `db:"scope_schema"`
	ScopeName              String `db:"scope_name"`
	MaximumCardinality     *int   `db:"maximum_cardinality"`
	DTDIdentifier          String `db:"dtd_identifier"`
}
