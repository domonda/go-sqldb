package sqldb

import (
	"context"
	"fmt"
	"strings"
)

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, conn Executor, queryBuilder QueryBuilder, table string, values Values, where string, args ...any) error {
	if len(values) == 0 {
		return fmt.Errorf("Update table %s: no values passed", table)
	}
	var query strings.Builder
	vals, err := queryBuilder.UpdateValues(&query, table, values, where, args)
	if err != nil {
		return fmt.Errorf("can't create UPDATE query because: %w", err)
	}
	err = conn.Exec(ctx, query.String(), vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query.String(), args, queryBuilder)
	}
	return nil
}
