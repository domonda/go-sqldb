package information

import (
	"context"
	"strings"

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
	return db.QueryRowValue[*Table](ctx,
		/*sql*/ `
			SELECT *
			FROM information_schema.tables
			WHERE table_catalog = $1
				AND table_schema = $2
				AND table_name = $3
		`,
		catalog, // $1
		schema,  // $2
		name,    // $3
	)
}

// TableExists checks if a table exists in the database
// qualifiedName is in the format "schema.table" or "table"
// If no schema is provided, "public" is assumed.
func TableExists(ctx context.Context, qualifiedName string) (exists bool, err error) {
	schema, table, ok := strings.Cut(qualifiedName, ".")
	if !ok {
		schema = "public"
		table = qualifiedName
	}
	return db.QueryRowValue[bool](ctx,
		/*sql*/ `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = $1
					AND table_name = $2
			)
		`,
		schema, // $1
		table,  // $2
	)
}

func GetAllTables(ctx context.Context) (tables []*Table, err error) {
	return db.QueryRowsAsSlice[*Table](ctx,
		/*sql*/ `
			SELECT * FROM information_schema.tables
		`,
	)
}
