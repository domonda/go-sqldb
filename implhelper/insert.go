package implhelper

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-wraperr"
)

// Insert a new row into table using the columnValues.
func Insert(ctx context.Context, conn sqldb.Connection, table string, columnValues sqldb.Values) error {
	if len(columnValues) == 0 {
		return fmt.Errorf("Insert into table %s: no columnValues", table)
	}

	names, values := sortedNamesAndValues(columnValues)
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	err := conn.ExecContext(ctx, query.String(), values...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}

	return nil
}

// InsertReturning inserts a new row into table using columnValues
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, conn sqldb.Connection, table string, columnValues sqldb.Values, returning string) sqldb.RowScanner {
	if len(columnValues) == 0 {
		return sqldb.NewErrRowScanner(fmt.Errorf("InsertReturning into table %s: no columnValues", table))
	}

	names, values := sortedNamesAndValues(columnValues)
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	query.WriteString(" RETURNING ")
	query.WriteString(returning)
	return conn.QueryRow(query.String(), values...)
}

// InsertStruct inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
func InsertStruct(ctx context.Context, conn sqldb.Connection, table string, rowStruct interface{}, ignoreColumns, restrictToColumns []string) error {
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

	names, values := structFields(v, ignoreColumns, restrictToColumns)
	if len(names) == 0 {
		return fmt.Errorf("InsertStruct into table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}
	var query strings.Builder
	writeInsertQuery(&query, table, names)

	err := conn.ExecContext(ctx, query.String(), values...)
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
func UpsertStruct(ctx context.Context, conn sqldb.Connection, table string, rowStruct interface{}, idColumn string, ignoreColumns, restrictToColumns []string) error {
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

	names, values := structFields(v, ignoreColumns, restrictToColumns)
	if len(names) == 0 {
		return fmt.Errorf("UpsertStruct to table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}
	var query strings.Builder
	writeInsertQuery(&query, table, names)
	fmt.Fprintf(&query, `ON CONFLICT("%s") DO UPDATE SET `, idColumn)
	first := true
	for i, name := range names {
		if name != idColumn {
			if first {
				first = false
			} else {
				query.WriteByte(',')
			}
			fmt.Fprintf(&query, `"%s"=$%d`, name, i+1)
		}
	}

	err := conn.ExecContext(ctx, query.String(), values...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}

func sortedNamesAndValues(columnValues sqldb.Values) (names []string, values []interface{}) {
	names = make([]string, 0, len(columnValues))
	for name := range columnValues {
		names = append(names, name)
	}
	sort.Strings(names)

	values = make([]interface{}, len(columnValues))
	for i, name := range names {
		values[i] = columnValues[name]
	}

	return names, values
}

func writeInsertQuery(w *strings.Builder, table string, names []string) {
	fmt.Fprintf(w, "INSERT INTO %s (", table)
	for i, name := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteByte('"')
		w.WriteString(name)
		w.WriteByte('"')
	}
	w.WriteString(") VALUES (")
	for i := range names {
		if i > 0 {
			w.WriteByte(',')
		}
		fmt.Fprintf(w, "$%d", i+1)
	}
	w.WriteByte(')')
}

func structFields(v reflect.Value, ignoreNames, restrictToNames []string) (names []string, values []interface{}) {
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		fieldValue := v.Field(i)
		switch {
		case fieldType.Anonymous:
			embedNames, embedValues := structFields(fieldValue, ignoreNames, restrictToNames)
			names = append(names, embedNames...)
			values = append(values, embedValues...)

		case fieldType.PkgPath == "":
			name := fieldType.Tag.Get("db")
			if validName(name, ignoreNames, restrictToNames) {
				names = append(names, name)
				values = append(values, fieldValue.Interface())
			}
		}
	}
	return names, values
}

func validName(name string, ignoreNames, restrictToNames []string) bool {
	if name == "" || name == "-" {
		return false
	}
	for _, ignore := range ignoreNames {
		if name == ignore {
			return false
		}
	}
	if len(restrictToNames) == 0 {
		return true
	}
	for _, allowed := range restrictToNames {
		if name == allowed {
			return true
		}
	}
	return false
}
