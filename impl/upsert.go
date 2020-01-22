package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-wraperr"
)

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// If inserting conflicts on pkColumn, then an update of the existing row is performed.
func UpsertStruct(ctx context.Context, conn sqldb.Connection, table, pkColumn string, rowStruct interface{}, namer sqldb.StructFieldNamer, ignoreColumns, restrictToColumns []string) error {
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

	pkColumns := strings.Split(pkColumn, ",")
	pkColumnsFound := make([]bool, len(pkColumns))

	var query strings.Builder
	writeInsertQuery(&query, table, names)
	fmt.Fprintf(&query, ` ON CONFLICT("%s") DO UPDATE SET `, strings.ReplaceAll(pkColumn, `,`, `","`))
	first := true
	for i, name := range names {
		if pki := indexOf(pkColumns, name); pki != -1 {
			pkColumnsFound[pki] = true
			continue
		}
		if first {
			first = false
		} else {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, `"%s"=$%d`, name, i+1)
	}
	for i, found := range pkColumnsFound {
		if !found {
			columns, _ := json.Marshal(names) // JSON array syntax is a nice format for the error
			return fmt.Errorf("UpsertStruct to table %s: pkColumn %q not found in columns %s", table, pkColumns[i], columns)
		}
	}

	err := conn.ExecContext(ctx, query.String(), vals...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", query.String(), err)
	}
	return nil
}

func indexOf(s []string, str string) int {
	for i, comp := range s {
		if comp == str {
			return i
		}
	}
	return -1
}
