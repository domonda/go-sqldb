package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values sqldb.Values) error {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
	)
	return sqldb.Insert(ctx, conn, queryBuilder, table, values)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
	)
	return sqldb.InsertUnique(ctx, conn, queryBuilder, table, values, onConflict)
}

// // InsertReturning inserts a new row into table using values
// // and returns values from the inserted row listed in returning.
// func InsertReturning(ctx context.Context, table string, values Values, returning string) sqldb.RowScanner {
// 	if len(values) == 0 {
// 		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
// 	}
// 	conn := Conn(ctx)

// 	var query strings.Builder
// 	names, vals := values.Sorted()
// 	err = writeInsert(&query, table, names, conn)
// 	query.WriteString(" RETURNING ")
// 	query.WriteString(returning)
// 	return conn.QueryRow(query.String(), vals...) // TODO wrap error with query
// }

// InsertRowStruct inserts a new row into table.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.InsertRowStruct(ctx, conn, queryBuilder, reflector, rowStruct, options...)
}

func InsertRowStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.InsertRowStructStmt[S](ctx, conn, queryBuilder, reflector, options...)
}

// func InsertStructStmt[S StructWithTableName](ctx context.Context, query string) (stmtFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
// 	conn := Conn(ctx)
// 	stmt, err := conn.Prepare(ctx, query)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	stmtFunc = func(ctx context.Context, rowStruct S) error {
// 		TODO
// 		if err != nil {
// 			return sqldb.WrapErrorWithQuery(err, query, args, conn)
// 		}
// 		return nil
// 	}
// 	return stmtFunc, stmt.Close, nil
// }

// InsertUniqueRowStruct inserts a new row with unique private key.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns true if a row was inserted.
func InsertUniqueRowStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, onConflict string, options ...sqldb.QueryOption) (inserted bool, err error) {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.InsertUniqueRowStruct(ctx, conn, queryBuilder, reflector, rowStruct, onConflict, options...)
}

// InsertRowStructs inserts a slice structs
// as new rows into table using the DefaultStructReflector.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	var (
		conn         = Conn(ctx)
		queryBuilder = QueryBuilderFuncFromContext(ctx)(conn)
		reflector    = GetStructReflector(ctx)
	)
	return sqldb.InsertRowStructs(ctx, conn, queryBuilder, reflector, rowStructs, options...)
}
