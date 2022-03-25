package impl

import (
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on pkColumn, then an update of the existing row is performed.
func UpsertStruct(conn sqldb.Connection, table string, rowStruct any, namer sqldb.StructFieldNamer, argFmt string, ignoreColumns, restrictToColumns []string) error {
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

	columns, flags, vals := structFieldValues(v, namer, ignoreColumns, restrictToColumns, true)
	if len(columns) == 0 {
		return fmt.Errorf("UpsertStruct to table %s: %T has no exported struct fields with `db` tag", table, rowStruct)
	}

	var b strings.Builder
	writeInsertQuery(&b, table, argFmt, columns)
	b.WriteString(` ON CONFLICT(`)
	hasPK := false
	for i := range columns {
		if !flags[i].IsPrimaryKey() {
			continue
		}
		if !hasPK {
			hasPK = true
		} else {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"%s"`, columns[i])
	}
	if !hasPK {
		return fmt.Errorf("UpsertStruct to table %s: %T has no exported struct fields with ,pk tag value suffix to mark primary key column(s)", table, rowStruct)
	}

	b.WriteString(`) DO UPDATE SET `)
	first := true
	for i := range columns {
		if f := flags[i]; f.IsPrimaryKey() || f.IsReadOnly() {
			continue
		}
		if first {
			first = false
		} else {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"%s"=$%d`, columns[i], i+1)
	}
	query := b.String()

	err := conn.Exec(query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}
