package sqldb

import (
	"context"
	"fmt"
	"strings"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, conn QueryExec, queryBuilder QueryBuilder, table string, values Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}
	query := strings.Builder{}
	cols, vals := values.SortedColumnsAndValues()
	err := queryBuilder.Insert(&query, table, cols, conn)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}
	return Exec(ctx, conn, query.String(), vals...)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, conn QueryExec, queryBuilder QueryBuilder, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	var query strings.Builder
	cols, vals := values.SortedColumnsAndValues()
	err = queryBuilder.InsertUnique(&query, table, cols, onConflict, conn)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	rows := conn.Query(ctx, query.String(), vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query.String(), vals, conn)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}
