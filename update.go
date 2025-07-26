package sqldb

import (
	"context"
	"fmt"
	"strings"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, conn QueryExec, queryBuilder QueryBuilder, table string, values Values, where string, args ...any) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}
	var query strings.Builder
	vals, err := queryBuilder.UpdateValues(&query, table, values, where, args, conn)
	if err != nil {
		return err
	}
	return Exec(ctx, conn, query.String(), vals...)
}
