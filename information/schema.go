package information

type Schema struct {
	CatalogName                String `db:"catalog_name"`
	SchemaName                 String `db:"schema_name"`
	SchemaOwner                String `db:"schema_owner"`
	DefaultCharacterSetCatalog String `db:"default_character_set_catalog"`
	DefaultCharacterSetSchema  String `db:"default_character_set_schema"`
	DefaultCharacterSetName    String `db:"default_character_set_name"`
	SqlPath                    String `db:"sql_path"`
}
