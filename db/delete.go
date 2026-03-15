package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// DeleteRowStruct deletes a row from the table identified by the primary key columns
// of the given struct. Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The struct must have at least one primary key field.
func DeleteRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName) error {
	conn := Conn(ctx)
	return sqldb.DeleteRowStruct(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStruct,
	)
}

// DeleteRowStructStmt prepares a statement for deleting rows of type S.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"id,primarykey"`).
// The struct must have at least one primary key field.
// Returns a delete function and a closeStmt function that must be called when done.
func DeleteRowStructStmt[S sqldb.StructWithTableName](ctx context.Context) (deleteFunc func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	conn := Conn(ctx)
	return sqldb.DeleteRowStructStmt[S](
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
	)
}

// DeleteRowStructs deletes a slice of structs within a transaction
// using a prepared statement.
// Table name, column names, and primary key columns are determined by
// the [StructReflector] from the context. The default reflector uses `db` struct tags
// (e.g., sqldb.TableName `db:"my_table"`, field `db:"id,primarykey"`).
func DeleteRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S) error {
	conn := Conn(ctx)
	return sqldb.DeleteRowStructs(
		ctx,
		conn,
		StructReflector(ctx),
		QueryBuilder(ctx),
		conn,
		rowStructs,
	)
}
