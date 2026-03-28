package sqliteconn

import (
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

var (
	_ sqldb.QueryBuilder          = (*QueryBuilder)(nil)
	_ sqldb.UpsertQueryBuilder    = (*QueryBuilder)(nil)
	_ sqldb.ReturningQueryBuilder = (*QueryBuilder)(nil)
)

// QueryBuilder implements [sqldb.QueryBuilder], [sqldb.UpsertQueryBuilder],
// and [sqldb.ReturningQueryBuilder] using SQLite-compatible syntax.
// It embeds [sqldb.StdReturningQueryBuilder] for standard CRUD and RETURNING operations
// and adds ON CONFLICT syntax for upserts.
//
// Unlike PostgreSQL's $N positional placeholders which can reuse arguments,
// SQLite uses sequential ? placeholders. This requires different strategies:
//   - Upsert uses excluded.column references instead of reusing placeholders
//   - Update places SET values before WHERE args to match SQL clause order
type QueryBuilder struct {
	sqldb.StdReturningQueryBuilder
}

// InsertUnique builds an INSERT query with ON CONFLICT DO NOTHING.
// The onConflict string specifies the conflict target columns.
func (b QueryBuilder) InsertUnique(formatter sqldb.QueryFormatter, table string, columns []sqldb.ColumnInfo, onConflict string) (query string, err error) {
	var q strings.Builder
	insert, err := b.Insert(formatter, table, columns)
	if err != nil {
		return "", err
	}
	q.WriteString(insert)
	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}
	fmt.Fprintf(&q, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)
	return q.String(), nil
}

// Upsert builds an INSERT ... ON CONFLICT DO UPDATE SET query.
// Primary key columns are used as the conflict target,
// non-primary key columns are updated on conflict using excluded.column
// references (since SQLite's sequential ? placeholders cannot reuse arguments
// like PostgreSQL's positional $N placeholders).
func (b QueryBuilder) Upsert(formatter sqldb.QueryFormatter, table string, columns []sqldb.ColumnInfo) (query string, err error) {
	hasNonPK := false
	for i := range columns {
		if !columns[i].PrimaryKey {
			hasNonPK = true
			break
		}
	}
	if !hasNonPK {
		return "", fmt.Errorf("Upsert requires at least one non-primary-key column")
	}

	var q strings.Builder
	insert, err := b.Insert(formatter, table, columns)
	if err != nil {
		return "", err
	}
	q.WriteString(insert)
	q.WriteString(` ON CONFLICT(`)
	first := true
	for i := range columns {
		if !columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			q.WriteByte(',')
		}
		columnName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		q.WriteString(columnName)
	}
	q.WriteString(`) DO UPDATE SET`)
	first = true
	for i := range columns {
		if columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			q.WriteByte(',')
		}
		columnName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&q, ` %s=excluded.%s`, columnName, columnName)
	}
	return q.String(), nil
}

// Update builds an UPDATE SET ... WHERE query with correct argument ordering
// for SQLite's sequential ? placeholders.
// Unlike [sqldb.StdQueryBuilder.Update] which returns whereArgs before values
// (correct for positional $N placeholders), this returns values before whereArgs
// to match the order of ? placeholders in the generated SQL (SET before WHERE).
func (b QueryBuilder) Update(formatter sqldb.QueryFormatter, table string, values sqldb.Values, where string, whereArgs []any) (query string, queryArgs []any, err error) {
	if len(values) == 0 {
		return "", nil, fmt.Errorf("Update table %s: no values passed", table)
	}
	tableName, err := formatter.FormatTableName(table)
	if err != nil {
		return "", nil, err
	}

	columns, vals := values.SortedColumnsAndValues()

	var q strings.Builder
	fmt.Fprintf(&q, `UPDATE %s SET`, tableName)
	for i := range columns {
		columnName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", nil, err
		}
		if i > 0 {
			q.WriteByte(',')
		}
		fmt.Fprintf(&q, ` %s=%s`, columnName, formatter.FormatPlaceholder(i))
	}
	fmt.Fprintf(&q, ` WHERE %s`, where)

	// For sequential ? placeholders, args must follow SQL order:
	// SET values first (appears first in SQL), then WHERE args.
	return q.String(), append(vals, whereArgs...), nil
}

// UpdateReturning builds an UPDATE SET ... WHERE ... RETURNING query
// using the SQLite-compatible [QueryBuilder.Update] argument ordering.
func (b QueryBuilder) UpdateReturning(formatter sqldb.QueryFormatter, table string, values sqldb.Values, returning, where string, whereArgs []any) (query string, queryArgs []any, err error) {
	query, queryArgs, err = b.Update(formatter, table, values, where, whereArgs)
	if err != nil {
		return "", nil, err
	}
	return query + " RETURNING " + returning, queryArgs, nil
}
