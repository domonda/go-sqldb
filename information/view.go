package information

type View struct {
	TableCatalog             String `db:"table_catalog"`
	TableSchema              String `db:"table_schema"`
	TableName                String `db:"table_name"`
	ViewDefinition           String `db:"view_definition"`
	CheckOption              String `db:"check_option"`
	IsUpdatable              YesNo  `db:"is_updatable"`
	IsInsertableInto         YesNo  `db:"is_insertable_into"`
	IsTriggerUpdatable       YesNo  `db:"is_trigger_updatable"`
	IsTriggerDeletable       YesNo  `db:"is_trigger_deletable"`
	IsTriggerInsertableInto  YesNo  `db:"is_trigger_insertable_into"`
}
