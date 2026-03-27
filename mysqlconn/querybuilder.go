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

// InsertUnique is not supported by MySQL because MySQL does not support
// the RETURNING clause needed to determine if a row was inserted.
func (QueryBuilder) InsertUnique(formatter sqldb.QueryFormatter, table string, columns []sqldb.ColumnInfo, onConflict string) (query string, err error) {
	return "", fmt.Errorf("mysqlconn.QueryBuilder: InsertUnique is not supported because MySQL has no RETURNING clause")
}

// Upsert builds an INSERT ... ON DUPLICATE KEY UPDATE query.
// Primary key columns are used to detect conflicts,
// non-primary key columns are updated using VALUES(col).
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
