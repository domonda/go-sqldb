package db

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
)

type Values = sqldb.Values

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}

	conn := Conn(ctx)
	names, vals := values.Sorted()
	b := strings.Builder{}
	writeInsertQuery(&b, table, conn, names)
	query := b.String()

	err := conn.Exec(query, vals...)

	return sqldb.WrapNonNilErrorWithQuery(err, query, conn, vals)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	conn := Conn(ctx)
	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, conn, names)
	fmt.Fprintf(&query, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)

	err = conn.QueryRow(query.String(), vals...).Scan(&inserted)

	err = sqldb.ReplaceErrNoRows(err, nil)
	err = sqldb.WrapNonNilErrorWithQuery(err, query.String(), conn, vals)
	return inserted, err
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, table string, values Values, returning string) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
	}

	conn := Conn(ctx)
	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, conn, names)
	query.WriteString(" RETURNING ")
	query.WriteString(returning)
	return conn.QueryRow(query.String(), vals...)
}

// InsertStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStruct(ctx context.Context, rowStruct any, ignoreColumns ...reflection.ColumnFilter) error {
	conn := Conn(ctx)
	mapper := conn.StructFieldMapper()

	table, columns, vals, err := insertStructValues(rowStruct, mapper, ignoreColumns)
	if err != nil {
		return err
	}

	var b strings.Builder
	writeInsertQuery(&b, table, conn, columns)
	query := b.String()

	err = conn.Exec(query, vals...)

	return sqldb.WrapNonNilErrorWithQuery(err, query, conn, vals)
}

// InsertUniqueStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns if a row was inserted.
func InsertUniqueStruct(ctx context.Context, rowStruct any, onConflict string, ignoreColumns ...reflection.ColumnFilter) (inserted bool, err error) {
	conn := Conn(ctx)
	mapper := conn.StructFieldMapper()

	table, columns, vals, err := insertStructValues(rowStruct, mapper, ignoreColumns)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	var b strings.Builder
	writeInsertQuery(&b, table, conn, columns)
	fmt.Fprintf(&b, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)
	query := b.String()

	err = conn.QueryRow(query, vals...).Scan(&inserted)
	err = sqldb.ReplaceErrNoRows(err, nil)

	return inserted, sqldb.WrapNonNilErrorWithQuery(err, query, conn, vals)
}

func writeInsertQuery(w *strings.Builder, table string, argFmt sqldb.ParamPlaceholderFormatter, names []string) {
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
		w.WriteString(argFmt.ParamPlaceholder(i))
	}
	w.WriteByte(')')
}

func insertStructValues(rowStruct any, mapper reflection.StructFieldMapper, ignoreColumns []reflection.ColumnFilter) (table string, columns []string, vals []any, err error) {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return "", nil, nil, fmt.Errorf("can't insert nil")
	case v.Kind() != reflect.Struct:
		return "", nil, nil, fmt.Errorf("expected struct but got %T", rowStruct)
	}

	table, columns, _, vals, err = reflection.ReflectStructValues(v, mapper, append(ignoreColumns, sqldb.IgnoreReadOnly))
	return table, columns, vals, err
}
