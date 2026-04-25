package information

// Schema maps a row from information_schema.schemata.
//
// Vendor support:
//   - PostgreSQL: all fields populated.
//   - MySQL/MariaDB: only SchemaName and the DefaultCharacterSet* fields
//     are populated; CatalogName is "def" and the rest scan as empty.
//   - SQL Server: only CatalogName, SchemaName, and SchemaOwner are
//     populated.
//   - SQLite, Oracle: information_schema is not implemented.
type Schema struct {
	CatalogName                String `db:"catalog_name"`
	SchemaName                 String `db:"schema_name"`
	SchemaOwner                String `db:"schema_owner"`
	DefaultCharacterSetCatalog String `db:"default_character_set_catalog"`
	DefaultCharacterSetSchema  String `db:"default_character_set_schema"`
	DefaultCharacterSetName    String `db:"default_character_set_name"`
	SqlPath                    String `db:"sql_path"`
}
