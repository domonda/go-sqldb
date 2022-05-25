package information

import (
	"context"

	"github.com/domonda/go-sqldb/db"
)

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

func GetTable(ctx context.Context, catalog, schema, name string) (table *Table, err error) {
	err = db.Conn(ctx).QueryRow(
		`select *
			from information_schema.tables
			where table_catalog = $1
				and table_schema = $2
				and table_name = $3`,
		catalog,
		schema,
		name,
	).ScanStruct(&table)
	if err != nil {
		return nil, err
	}
	return table, nil
}

func GetAllTables(ctx context.Context) (tables []*Table, err error) {
	err = db.Conn(ctx).QueryRows(
		`select * from information_schema.tables`,
	).ScanStructSlice(&tables)
	if err != nil {
		return nil, err
	}
	return tables, nil
}
