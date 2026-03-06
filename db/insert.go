package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values sqldb.Values) error {
	conn := Conn(ctx)
	return sqldb.Insert(ctx, conn, conn, conn, table, values)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	conn := Conn(ctx)
	return sqldb.InsertUnique(ctx, conn, conn, conn, table, values, onConflict)
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, table string, values sqldb.Values, returning string) *sqldb.Row {
	conn := Conn(ctx)
	return sqldb.InsertReturning(ctx, conn, conn, conn, conn, table, values, returning)
}

// InsertRowStruct inserts a new row into the table for the given struct.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	return sqldb.InsertRowStruct(ctx, conn, conn, conn, conn, rowStruct, options...)
}

// InsertRowStructStmt prepares a statement for inserting rows of type S.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Returns an insertFunc to insert individual rows and a closeStmt
// function that must be called when done.
func InsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.InsertRowStructStmt[S](ctx, conn, conn, conn, conn, options...)
}

// InsertUniqueRowStruct inserts a new row or does nothing if the onConflict statement applies.
// Returns true if a row was inserted.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertUniqueRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, onConflict string, options ...sqldb.QueryOption) (inserted bool, err error) {
	conn := Conn(ctx)
	return sqldb.InsertUniqueRowStruct(ctx, conn, conn, conn, conn, rowStruct, onConflict, options...)
}

// InsertRowStructs inserts a slice of structs as new rows into the table for the given struct type.
// The table name is derived from the `db` struct tag of an embedded sqldb.TableName field
// (e.g., sqldb.TableName `db:"my_table"`).
// Column names are derived from the `db` struct tags of the struct's fields.
// Optional QueryOption can be passed to ignore mapped columns.
func InsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	conn := Conn(ctx)
	return sqldb.InsertRowStructs(ctx, conn, conn, conn, conn, rowStructs, options...)
}
