package db

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/domonda/go-sqldb"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, table string, values sqldb.Values, where string, args ...any) error {
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)
	return sqldb.Update(ctx, conn, queryBuilder, table, values, where, args...)
}

// // UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
// // and returning a single row with the columns specified in returning argument.
// func UpdateReturningRow(ctx context.Context, table string, values Values, returning, where string, args ...any) sqldb.RowScanner {
// 	if len(values) == 0 {
// 		return sqldb.RowScannerWithError(fmt.Errorf("UpdateReturningRow table %s: no values passed", table))
// 	}
// 	conn := Conn(ctx)

// 	query, vals := buildUpdateQuery(table, values, where, args, conn)
// 	query += " RETURNING " + returning
// 	return conn.QueryRow(query, vals...)
// }

// // UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
// // and returning multiple rows with the columns specified in returning argument.
// func UpdateReturningRows(ctx context.Context, table string, values Values, returning, where string, args ...any) sqldb.RowsScanner {
// 	if len(values) == 0 {
// 		return sqldb.RowsScannerWithError(fmt.Errorf("UpdateReturningRows table %s: no values passed", table))
// 	}
// 	conn := Conn(ctx)

// 	query, vals := buildUpdateQuery(table, values, where, args, conn)
// 	query += " RETURNING " + returning
// 	return conn.QueryRows(query, vals...)
// }

// UpdateStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
func UpdateStruct(ctx context.Context, table string, rowStruct any, options ...QueryOption) error {
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)

	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return fmt.Errorf("UpdateStruct of table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("UpdateStruct of table %s: expected struct but got %T", table, rowStruct)
	}

	columns, vals := ReflectStructColumnsAndValues(v, defaultStructReflector, append(options, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col sqldb.ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpdateStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var query strings.Builder
	err := queryBuilder.UpdateColumns(&query, table, columns)
	if err != nil {
		return err
	}
	return sqldb.Exec(ctx, conn, query.String(), vals...)
}
