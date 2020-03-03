package impl

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-wraperr"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, conn sqldb.Connection, table string, values sqldb.Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}

	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}

	return nil
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, conn sqldb.Connection, table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	fmt.Fprintf(&query, " ON CONFLICT %s DO NOTHING RETURNING TRUE", onConflict)

	err = conn.QueryRow(query.String(), vals...).Scan(&inserted)
	return inserted, sqldb.RemoveErrNoRows(err)
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, conn sqldb.Connection, table string, values sqldb.Values, returning string) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
	}

	names, vals := values.Sorted()
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	query.WriteString(" RETURNING ")
	query.WriteString(returning)
	return conn.QueryRow(query.String(), vals...)
}

func writeInsertQuery(w *strings.Builder, table string, names []string) {
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
		fmt.Fprintf(w, "$%d", i+1)
	}
	w.WriteByte(')')
}

// InsertStruct inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
func InsertStruct(ctx context.Context, conn sqldb.Connection, table string, rowStruct interface{}, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) error {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return fmt.Errorf("InsertStruct into table %s: can't upsert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("InsertStruct into table %s: expected struct but got %T", table, rowStruct)
	}

	columns, _, vals := structFields(v, namer, ignoreColumns, restrictToColumns, false)
	if len(columns) == 0 {
		return fmt.Errorf("InsertStruct into table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}
	var query strings.Builder
	writeInsertQuery(&query, table, columns)

	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}
