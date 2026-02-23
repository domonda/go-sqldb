package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values sqldb.Values) error {
	return sqldb.Insert(ctx, Conn(ctx), table, values)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return sqldb.InsertUnique(ctx, Conn(ctx), table, values, onConflict)
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, table string, values sqldb.Values, returning string) *sqldb.Row {
	return sqldb.InsertReturning(ctx, Conn(ctx), table, values, returning)
}

// InsertRowStruct inserts a new row into the table for the given struct.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	return sqldb.InsertRowStruct(ctx, Conn(ctx), rowStruct, options...)
}

// InsertRowStructStmt prepares a statement for inserting rows of type S.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Returns an insertFunc to insert individual rows and a closeStmt
// function that must be called when done.
func InsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	return sqldb.InsertRowStructStmt[S](ctx, Conn(ctx), options...)
}

// InsertUniqueRowStruct inserts a new row or does nothing if the onConflict statement applies.
// Returns true if a row was inserted.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertUniqueRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, onConflict string, options ...sqldb.QueryOption) (inserted bool, err error) {
	return sqldb.InsertUniqueRowStruct(ctx, Conn(ctx), rowStruct, onConflict, options...)
}

// InsertRowStructs inserts a slice of structs as new rows into the table for the given struct type.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	return sqldb.InsertRowStructs(ctx, Conn(ctx), rowStructs, options...)
}
