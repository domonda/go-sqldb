package db

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values sqldb.Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}
	conn := Conn(ctx)

	var query strings.Builder
	names, vals := values.Sorted()
	writeInsertQuery(&query, table, names, conn)

	err := conn.Exec(query.String(), vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return nil
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}
	conn := Conn(ctx)

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	var query strings.Builder
	names, vals := values.Sorted()
	writeInsertQuery(&query, table, names, conn)
	fmt.Fprintf(&query, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)

	inserted, err = QueryValue[bool](ctx, query.String(), vals...)
	err = sqldb.ReplaceErrNoRows(err, nil)
	if err != nil {
		return false, wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return inserted, err
}

// // InsertReturning inserts a new row into table using values
// // and returns values from the inserted row listed in returning.
// func InsertReturning(ctx context.Context, table string, values sqldb.Values, returning string) sqldb.RowScanner {
// 	if len(values) == 0 {
// 		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
// 	}
// 	conn := Conn(ctx)

// 	var query strings.Builder
// 	names, vals := values.Sorted()
// 	writeInsertQuery(&query, table, names, conn)
// 	query.WriteString(" RETURNING ")
// 	query.WriteString(returning)
// 	return conn.QueryRow(query.String(), vals...) // TODO wrap error with query
// }

// InsertStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStruct(ctx context.Context, table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	conn := Conn(ctx)
	columns, vals, err := insertStructValues(table, rowStruct, conn.StructReflector(), ignoreColumns)
	if err != nil {
		return err
	}

	var query strings.Builder
	writeInsertQuery(&query, table, columns, conn)

	err = conn.Exec(query.String(), vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return nil
}

// InsertStructWithTableName inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStructWithTableName(ctx context.Context, row sqldb.StructWithTableName, ignoreColumns ...sqldb.ColumnFilter) error {
	table, err := Conn(ctx).StructReflector().TableNameForStruct(reflect.TypeOf(row))
	if err != nil {
		return err
	}
	return InsertStruct(ctx, table, row, ignoreColumns...)
}

// InsertUniqueStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns if a row was inserted.
func InsertUniqueStruct(ctx context.Context, table string, rowStruct any, onConflict string, ignoreColumns ...sqldb.ColumnFilter) (inserted bool, err error) {
	conn := Conn(ctx)
	columns, vals, err := insertStructValues(table, rowStruct, conn.StructReflector(), ignoreColumns)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	var query strings.Builder
	writeInsertQuery(&query, table, columns, conn)
	fmt.Fprintf(&query, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)

	inserted, err = QueryValue[bool](ctx, query.String(), vals...)
	err = sqldb.ReplaceErrNoRows(err, nil)
	if err != nil {
		return false, wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return inserted, err
}

// InsertStructs inserts a slice or array of structs
// as new rows into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
//
// TODO optimized version with single query if possible
// split into multiple queries depending or maxArgs for query
func InsertStructs(ctx context.Context, table string, rowStructs any, ignoreColumns ...sqldb.ColumnFilter) error {
	v := reflect.ValueOf(rowStructs)
	if k := v.Type().Kind(); k != reflect.Slice && k != reflect.Array {
		return fmt.Errorf("InsertStructs expects a slice or array as rowStructs, got %T", rowStructs)
	}
	numRows := v.Len()
	return Transaction(ctx, func(ctx context.Context) error {
		for i := 0; i < numRows; i++ {
			err := InsertStruct(ctx, table, v.Index(i).Interface(), ignoreColumns...)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func writeInsertQuery(w *strings.Builder, table string, names []string, format sqldb.PlaceholderFormatter) {
	fmt.Fprintf(w, `INSERT INTO %s(`, table)
	for i, name := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteByte('"')
		w.WriteString(name)
		w.WriteByte('"')
	}
	w.WriteString(`) VALUES(`)
	for i := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteString(format.Placeholder(i))
	}
	w.WriteByte(')')
}

func insertStructValues(table string, rowStruct any, namer sqldb.StructReflector, ignoreColumns []sqldb.ColumnFilter) (columns []string, vals []any, err error) {
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

	columns, _, vals = ReflectStructValues(v, namer, append(ignoreColumns, sqldb.IgnoreReadOnly))
	return columns, vals, nil
}
