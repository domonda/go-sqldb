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
	return db.QueryRowStruct[*Table](ctx,
		/*sql*/ `
			SELECT *
			FROM information_schema.tables
			WHERE table_catalog = $1
				AND table_schema = $2
				AND table_name = $3
		`,
		catalog,
		schema,
		name,
	)
}

func GetAllTables(ctx context.Context) (tables []*Table, err error) {
	return db.QuerySlice[*Table](ctx,
		/*sql*/ `
			SELECT * FROM information_schema.tables
		`,
	)
}
