package impl

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
)

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on pkColumn, then an update of the existing row is performed.
func UpsertStruct(ctx context.Context, conn Execer, table string, rowStruct any, mapper sqldb.StructFieldMapper, argFmt string, ignoreColumns []sqldb.ColumnFilter) error {
	v := reflect.ValueOf(rowStruct)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	switch {
	case v.Kind() == reflect.Pointer && v.IsNil():
		return fmt.Errorf("UpsertStruct to table %s: can't insert nil", table)
	case v.Kind() != reflect.Struct:
		return fmt.Errorf("UpsertStruct to table %s: expected struct but got %T", table, rowStruct)
	}

	columns, pkCols, vals := ReflectStructValues(v, mapper, append(ignoreColumns, sqldb.IgnoreReadOnly))
	if len(pkCols) == 0 {
		return fmt.Errorf("UpsertStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var b strings.Builder
	writeInsertQuery(&b, table, argFmt, columns)
	b.WriteString(` ON CONFLICT(`)
	for i, pkCol := range pkCols {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"%s"`, columns[pkCol])
	}

	b.WriteString(`) DO UPDATE SET `)
	first := true
	for i := range columns {
		if slices.Contains(pkCols, i) {
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

	_, err := conn.ExecContext(ctx, query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}
