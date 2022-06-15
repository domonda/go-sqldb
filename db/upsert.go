package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
	"golang.org/x/exp/slices"
)

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, rowStruct any, ignoreColumns ...reflection.ColumnFilter) error {
	v, err := derefStruct(rowStruct)
	if err != nil {
		return err
	}

	conn := Conn(ctx)
	argFmt := conn.ArgFmt()
	mapper := conn.StructFieldMapper()
	table, columns, pkCols, vals, err := reflection.ReflectStructValues(v, mapper, append(ignoreColumns, sqldb.IgnoreReadOnly))
	if err != nil {
		return err
	}
	if table == "" {
		return fmt.Errorf("UpsertStruct: %s has no table name", v.Type())
	}
	if len(pkCols) == 0 {
		return fmt.Errorf("UpsertStruct: %s has no mapped primary key field", v.Type())
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

	err = conn.Exec(query, vals...)

	return WrapNonNilErrorWithQuery(err, query, argFmt, vals)
}
