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
// and adds ON CONFLICT syntax for upserts (same as PostgreSQL).
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
		fmt.Fprintf(&q, ` %s=%s`, columnName, formatter.FormatPlaceholder(i))
	}
	return q.String(), nil
}
