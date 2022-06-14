package impl

import (
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(conn sqldb.Connection, table, argFmt string, values sqldb.Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}

	names, vals := values.Sorted()
	b := strings.Builder{}
	writeInsertQuery(&b, table, argFmt, names)
	query := b.String()

	err := conn.Exec(query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(conn sqldb.Connection, table, argFmt string, values sqldb.Values, onConflict string) (inserted bool, err error) {
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

	err = conn.QueryRow(query.String(), vals...).Scan(&inserted)

	err = sqldb.ReplaceErrNoRows(err, nil)
	err = WrapNonNilErrorWithQuery(err, query.String(), argFmt, vals)
	return inserted, err
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(conn sqldb.Connection, table, argFmt string, values sqldb.Values, returning string) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
	}

	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, argFmt, names)
	query.WriteString(" RETURNING ")
	query.WriteString(returning)
	return conn.QueryRow(query.String(), vals...)
}

func writeInsertQuery(w *strings.Builder, table, argFmt string, names []string) {
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
		fmt.Fprintf(w, argFmt, i+1)
	}
	w.WriteByte(')')
}

// InsertStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStruct(conn sqldb.Connection, table string, rowStruct any, namer sqldb.StructFieldMapper, argFmt string, ignoreColumns []sqldb.ColumnFilter) error {
	columns, vals, err := insertStructValues(table, rowStruct, namer, ignoreColumns)
	if err != nil {
		return err
	}

	var b strings.Builder
	writeInsertQuery(&b, table, argFmt, columns)
	query := b.String()

	err = conn.Exec(query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}

// InsertUniqueStruct inserts a new row into table using the connection's
// StructFieldMapper to map struct fields to column names.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns if a row was inserted.
func InsertUniqueStruct(conn sqldb.Connection, table string, rowStruct any, onConflict string, namer sqldb.StructFieldMapper, argFmt string, ignoreColumns []sqldb.ColumnFilter) (inserted bool, err error) {
	columns, vals, err := insertStructValues(table, rowStruct, namer, ignoreColumns)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	var b strings.Builder
	writeInsertQuery(&b, table, argFmt, columns)
	fmt.Fprintf(&b, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)
	query := b.String()

	err = conn.QueryRow(query, vals...).Scan(&inserted)
	err = sqldb.ReplaceErrNoRows(err, nil)

	return inserted, WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}

func insertStructValues(table string, rowStruct any, namer sqldb.StructFieldMapper, ignoreColumns []sqldb.ColumnFilter) (columns []string, vals []any, err error) {
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
