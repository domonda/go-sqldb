package implhelper

import (
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// InsertStruct inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If optional onlyColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be inserted.
func InsertStruct(conn sqldb.Connection, table string, rowStruct interface{}, onlyColumns ...string) error {
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

	names, values := structFields(v, onlyColumns)
	if len(names) == 0 {
		return fmt.Errorf("InsertStruct into table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}

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
	for i := range values {
		if i > 0 {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, "$%d", i+1)
	}
	query.WriteByte(')')

	// fmt.Println(query)

	return conn.Exec(query.String(), values...)
}

func structFields(v reflect.Value, allowedNames []string) (names []string, values []interface{}) {
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		fieldValue := v.Field(i)
		switch {
		case fieldType.Anonymous:
			embedNames, embedValues := structFields(fieldValue, allowedNames)
			names = append(names, embedNames...)
			values = append(values, embedValues...)

		case fieldType.PkgPath == "":
			name := fieldType.Tag.Get("db")
			if validName(name, allowedNames) {
				names = append(names, name)
				values = append(values, fieldValue.Interface())
			}
		}
	}
	return names, values
}

func validName(name string, allowedNames []string) bool {
	if name == "" || name == "-" {
		return false
	}
	if len(allowedNames) == 0 {
		return true
	}
	for _, allowedName := range allowedNames {
		if name == allowedName {
			return true
		}
	}
	return false
}
