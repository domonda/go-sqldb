package mysqlconn

import (
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

var _ sqldb.QueryBuilder = (*QueryBuilder)(nil)
var _ sqldb.UpsertQueryBuilder = (*QueryBuilder)(nil)

// QueryBuilder implements [sqldb.QueryBuilder] and [sqldb.UpsertQueryBuilder]
// using MySQL-specific syntax.
// It embeds [sqldb.StdQueryBuilder] for standard SQL operations
// and overrides upsert methods with MySQL ON DUPLICATE KEY UPDATE syntax.
// It does not implement [sqldb.ReturningQueryBuilder] because
// MySQL does not support the RETURNING clause.
type QueryBuilder struct {
	sqldb.StdQueryBuilder
}

// Update overrides StdQueryBuilder.Update to reorder query arguments
// for MySQL's positional ? placeholders.
// StdQueryBuilder.Update returns queryArgs as [whereArgs..., vals...]
// but the generated SQL has SET (vals) before WHERE (whereArgs).
// This override reorders to [vals..., whereArgs...] to match SQL order.
func (b QueryBuilder) Update(formatter sqldb.QueryFormatter, table string, values sqldb.Values, where string, whereArgs []any) (query string, queryArgs []any, err error) {
	query, queryArgs, err = b.StdQueryBuilder.Update(formatter, table, values, where, whereArgs)
	if err != nil {
		return "", nil, err
	}
	nWhere := len(whereArgs)
	if nWhere > 0 && nWhere < len(queryArgs) {
		reordered := make([]any, len(queryArgs))
		copy(reordered, queryArgs[nWhere:])
		copy(reordered[len(queryArgs)-nWhere:], queryArgs[:nWhere])
		queryArgs = reordered
	}
	return query, queryArgs, nil
}

// InsertUnique builds an INSERT ... ON DUPLICATE KEY UPDATE query
// that performs a no-op update on conflict, so the row is not modified.
// The number of affected rows reported by MySQL is 1 for an insert
// and 0 for a no-op update, allowing the caller to detect whether
// a row was inserted via ExecRowsAffected.
func (b QueryBuilder) InsertUnique(formatter sqldb.QueryFormatter, table string, columns []sqldb.ColumnInfo, onConflict string) (query string, err error) {
	insert, err := b.Insert(formatter, table, columns)
	if err != nil {
		return "", err
	}
	conflictCols := strings.Split(onConflict, ",")
	col := strings.TrimSpace(conflictCols[0])
	col, err = formatter.FormatColumnName(col)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s ON DUPLICATE KEY UPDATE %s = %s", insert, col, col), nil
}

// Upsert builds an INSERT ... ON DUPLICATE KEY UPDATE query.
// Primary key columns are used to detect conflicts,
// non-primary key columns are updated using VALUES(col) syntax
// for compatibility with both MySQL and MariaDB.
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
	q.WriteString(` ON DUPLICATE KEY UPDATE`)
	first := true
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
		fmt.Fprintf(&q, ` %s=VALUES(%s)`, columnName, columnName)
	}
	return q.String(), nil
}
