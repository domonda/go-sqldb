package sqldb

import (
	"context"
	"fmt"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, conn Executor, queryBuilder QueryBuilder, table string, values Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}
	cols, vals := values.SortedColumnsAndValues()
	query, err := queryBuilder.Insert(table, cols)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}
	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, queryBuilder)
	}
	return nil
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, conn Querier, queryBuilder QueryBuilder, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}

	cols, vals := values.SortedColumnsAndValues()
	query, err := queryBuilder.InsertUnique(table, cols, onConflict)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	rows := conn.Query(ctx, query, vals...)
	defer rows.Close()
	if err = rows.Err(); err != nil {
		return false, WrapErrorWithQuery(err, query, vals, queryBuilder)
	}
	// If there is a row returned, then a row was inserted.
	// The content of the returned row is not relevant.
	return rows.Next(), nil
}
