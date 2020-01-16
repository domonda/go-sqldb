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
	query := insertQuery(table, names, "")
	err := conn.ExecContext(ctx, query, values...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query, err)
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
	query := insertQuery(table, names, returning)
	return conn.QueryRow(query, values...)
}

// InsertStructContext inserts a new row into table using the exported fields
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
		return fmt.Errorf("InsertStruct into table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("InsertStruct into table %s: expected struct but got %T", table, rowStruct)
	}

	names, values := structFields(v, ignoreColumns, restrictToColumns)
	if len(names) == 0 {
		return fmt.Errorf("InsertStruct into table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}

	query := insertQuery(table, names, "")
	err := conn.ExecContext(ctx, query, values...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query, err)
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

func insertQuery(table string, names []string, returning string) string {
	var query strings.Builder
	fmt.Fprintf(&query, "INSERT INTO %s (", table)
	for i, name := range names {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteByte('"')
		query.WriteString(name)
		query.WriteByte('"')
	}
	query.WriteString(") VALUES (")
	for i := range names {
		if i > 0 {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, "$%d", i+1)
	}
	query.WriteByte(')')

	if returning != "" {
		query.WriteString(" RETURNING ")
		query.WriteString(returning)
	}

	return query.String()
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
