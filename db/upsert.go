package db

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
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

	columns, pkCols, vals := impl.ReflectStructValues(v, namer, append(ignoreColumns, sqldb.IgnoreReadOnly))
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

	conn := Conn(ctx)

	err := conn.Exec(query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}
