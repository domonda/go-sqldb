package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-wraperr"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, conn sqldb.Connection, table string, values sqldb.Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}

	names, vals := sortedNamesAndValues(values)
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}

	return nil
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, conn sqldb.Connection, table string, values sqldb.Values, returning string) sqldb.RowScanner {
	if len(values) == 0 {
		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
	}

	names, vals := sortedNamesAndValues(values)
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	query.WriteString(" RETURNING ")
	query.WriteString(returning)
	return conn.QueryRow(query.String(), vals...)
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

	names, vals := structFields(v, namer, ignoreColumns, restrictToColumns)
	if len(names) == 0 {
		return fmt.Errorf("InsertStruct into table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}
	var query strings.Builder
	writeInsertQuery(&query, table, names)

	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on idColumn, then an update of the existing row is performed.
func UpsertStruct(ctx context.Context, conn sqldb.Connection, table string, rowStruct interface{}, namer sqldb.StructFieldNamer, idColumn string, ignoreColumns, restrictToColumns []string) error {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Ptr && v.IsNil():
		return fmt.Errorf("UpsertStruct to table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("UpsertStruct to table %s: expected struct but got %T", table, rowStruct)
	}

	names, vals := structFields(v, namer, ignoreColumns, restrictToColumns)
	if len(names) == 0 {
		return fmt.Errorf("UpsertStruct to table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	fmt.Fprintf(&query, ` ON CONFLICT("%s") DO UPDATE SET `, idColumn)
	first := true
	idColumnFound := false
	for i, name := range names {
		if name == idColumn {
			idColumnFound = true
			continue
		}
		if first {
			first = false
		} else {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, `"%s"=$%d`, name, i+1)
	}
	if !idColumnFound {
		columns, _ := json.Marshal(names) // JSON array syntax is a nice format for the error
		return fmt.Errorf("UpsertStruct to table %s: idColumn %q not found in columns %s", table, idColumn, columns)
	}

	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}

func sortedNamesAndValues(values sqldb.Values) (names []string, vals []interface{}) {
	names = make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	vals = make([]interface{}, len(values))
	for i, name := range names {
		vals[i] = values[name]
	}

	return names, vals
}

func writeInsertQuery(w *strings.Builder, table string, names []string) {
	fmt.Fprintf(w, "INSERT INTO %s(", table)
	for i, name := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteByte('"')
		w.WriteString(name)
		w.WriteByte('"')
	}
	w.WriteString(") VALUES(")
	for i := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		fmt.Fprintf(w, "$%d", i+1)
	}
	w.WriteByte(')')
}

func structFields(v reflect.Value, namer sqldb.StructFieldNamer, ignoreNames, restrictToNames []string) (names []string, vals []interface{}) {
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		fieldValue := v.Field(i)
		switch {
		case fieldType.Anonymous:
			embedNames, embedValues := structFields(fieldValue, namer, ignoreNames, restrictToNames)
			names = append(names, embedNames...)
			vals = append(vals, embedValues...)

		case fieldType.PkgPath == "":
			name := namer.StructFieldName(fieldType)
			if validName(name, ignoreNames, restrictToNames) {
				names = append(names, name)
				vals = append(vals, fieldValue.Interface())
			}
		}
	}
	return names, vals
}
