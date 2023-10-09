package impl

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, conn Execer, table string, values sqldb.Values, converter driver.ValueConverter, argFmt string) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}

	names, args := values.Sorted()
	query := strings.Builder{}
	writeInsertQuery(&query, table, argFmt, names)

	return Exec(ctx, conn, query.String(), args, converter, argFmt)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, conn Queryer, table string, values sqldb.Values, onConflict string, converter driver.ValueConverter, argFmt string, mapper sqldb.StructFieldMapper) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, argFmt, names)
	fmt.Fprintf(&query, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)

	err = QueryRow(ctx, conn, query.String(), vals, converter, argFmt, mapper).Scan(&inserted)
	return inserted, sqldb.ReplaceErrNoRows(err, nil)
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, conn Queryer, table string, values sqldb.Values, returning string, converter driver.ValueConverter, argFmt string, mapper sqldb.StructFieldMapper) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
	}

	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, argFmt, names)
	query.WriteString(" RETURNING ")
	query.WriteString(returning)

	return QueryRow(ctx, conn, query.String(), vals, converter, argFmt, mapper)
}

// InsertStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStruct(ctx context.Context, conn Execer, table string, rowStruct any, mapper sqldb.StructFieldMapper, ignoreColumns []sqldb.ColumnFilter, converter driver.ValueConverter, argFmt string) error {
	columns, vals, err := insertStructValues(table, rowStruct, mapper, ignoreColumns)
	if err != nil {
		return err
	}

	var query strings.Builder
	writeInsertQuery(&query, table, argFmt, columns)

	return Exec(ctx, conn, query.String(), vals, converter, argFmt)
}

// InsertUniqueStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns if a row was inserted.
func InsertUniqueStruct(ctx context.Context, conn Queryer, mapper sqldb.StructFieldMapper, table string, rowStruct any, onConflict string, ignoreColumns []sqldb.ColumnFilter, converter driver.ValueConverter, argFmt string) (inserted bool, err error) {
	columns, vals, err := insertStructValues(table, rowStruct, mapper, ignoreColumns)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	var query strings.Builder
	writeInsertQuery(&query, table, argFmt, columns)
	fmt.Fprintf(&query, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)

	err = QueryRow(ctx, conn, query.String(), vals, converter, argFmt, mapper).Scan(&inserted)
	return inserted, sqldb.ReplaceErrNoRows(err, nil)
}

func insertStructValues(table string, rowStruct any, mapper sqldb.StructFieldMapper, ignoreColumns []sqldb.ColumnFilter) (columns []string, vals []any, err error) {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return nil, nil, fmt.Errorf("InsertStruct into table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return nil, nil, fmt.Errorf("InsertStruct into table %s: expected struct but got %T", table, rowStruct)
	}

	columns, _, vals = ReflectStructValues(v, mapper, append(ignoreColumns, sqldb.IgnoreReadOnly))
	return columns, vals, nil
}

// InsertStructs is a helper that calls InsertStruct for every
// struct in the rowStructs slice or array.
// The inserts are performed within a new transaction
// if the passed conn is not already a transaction.
func InsertStructs(conn sqldb.Connection, table string, rowStructs any, ignoreColumns ...sqldb.ColumnFilter) error {
	// TODO optimized version with single query if possible, split into multiple queries depending or maxArgs for query
	v := reflect.ValueOf(rowStructs)
	if k := v.Type().Kind(); k != reflect.Slice && k != reflect.Array {
		return fmt.Errorf("InsertStructs expects a slice or array as rowStructs, got %T", rowStructs)
	}
	numRows := v.Len()
	return sqldb.Transaction(conn, nil, func(tx sqldb.Connection) error {
		for i := 0; i < numRows; i++ {
			err := tx.InsertStruct(table, v.Index(i).Interface(), ignoreColumns...)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
