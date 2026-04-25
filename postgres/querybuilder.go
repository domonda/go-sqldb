// Package postgres provides a [QueryBuilder] implementing
// PostgreSQL/SQLite-compatible ON CONFLICT upsert syntax.
// It lives in the root module (github.com/domonda/go-sqldb)
// so that packages within the same module (like db) can use it
// without importing driver-specific modules like pqconn.
package postgres

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
// and [sqldb.ReturningQueryBuilder] using PostgreSQL/SQLite-compatible syntax.
// It embeds [sqldb.StdReturningQueryBuilder] for standard CRUD and RETURNING
// operations and adds ON CONFLICT syntax for upserts.
type QueryBuilder struct {
	sqldb.StdReturningQueryBuilder
}

// InsertUnique builds an INSERT query with ON CONFLICT DO NOTHING.
//
// conflictTarget is a comma-separated list of column names identifying
// the conflict target (optionally surrounded by a single pair of outer
// parentheses, which are stripped). It must NOT include the
// `ON CONFLICT` keyword: the builder emits `ON CONFLICT (cols) DO NOTHING`
// around the column list. See [sqldb.UpsertQueryBuilder] for the full
// contract and security model.
func (b QueryBuilder) InsertUnique(formatter sqldb.QueryFormatter, table string, columns []sqldb.ColumnInfo, conflictTarget string) (query string, err error) {
	var q strings.Builder
	insert, err := b.Insert(formatter, table, columns)
	if err != nil {
		return "", err
	}
	q.WriteString(insert)
	if strings.HasPrefix(conflictTarget, "(") && strings.HasSuffix(conflictTarget, ")") {
		conflictTarget = conflictTarget[1 : len(conflictTarget)-1]
	}
	fmt.Fprintf(&q, " ON CONFLICT (%s) DO NOTHING", conflictTarget)
	return q.String(), nil
}

// Upsert builds an INSERT ... ON CONFLICT DO UPDATE SET query.
// Primary key columns are used as the conflict target,
// non-primary key columns are updated on conflict.
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
	q.WriteString(` ON CONFLICT (`)
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
		fmt.Fprintf(&q, ` %s=%s`, columnName, formatter.FormatPlaceholder(i))
	}
	return q.String(), nil
}
