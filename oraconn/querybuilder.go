package oraconn

import (
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

var _ sqldb.QueryBuilder = (*QueryBuilder)(nil)
var _ sqldb.UpsertQueryBuilder = (*QueryBuilder)(nil)

// QueryBuilder implements [sqldb.QueryBuilder] and [sqldb.UpsertQueryBuilder]
// using Oracle-specific syntax.
// It embeds [sqldb.StdQueryBuilder] for standard SQL operations
// and overrides upsert methods with Oracle MERGE syntax.
// It does not implement [sqldb.ReturningQueryBuilder].
type QueryBuilder struct {
	sqldb.StdQueryBuilder
}

// InsertUnique builds a MERGE statement that inserts a row only if it
// does not conflict on the specified columns.
// The number of rows affected is 1 for an insert and 0 for a conflict.
func (b QueryBuilder) InsertUnique(formatter sqldb.QueryFormatter, table string, columns []sqldb.ColumnInfo, onConflict string) (query string, err error) {
	conflictCols := strings.Split(onConflict, ",")
	for i := range conflictCols {
		conflictCols[i] = strings.TrimSpace(conflictCols[i])
	}

	return b.buildMerge(formatter, table, columns, conflictCols, false)
}

// Upsert builds a MERGE statement that inserts a new row or updates
// an existing one when the primary key columns conflict.
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

	var pkCols []string
	for i := range columns {
		if columns[i].PrimaryKey {
			pkCols = append(pkCols, columns[i].Name)
		}
	}

	return b.buildMerge(formatter, table, columns, pkCols, true)
}

// buildMerge generates a MERGE INTO ... USING ... statement.
// If withUpdate is true, a WHEN MATCHED THEN UPDATE clause is added
// for non-primary-key columns.
func (QueryBuilder) buildMerge(formatter sqldb.QueryFormatter, table string, columns []sqldb.ColumnInfo, conflictCols []string, withUpdate bool) (string, error) {
	fmtTable, err := formatter.FormatTableName(table)
	if err != nil {
		return "", err
	}

	var q strings.Builder

	// MERGE INTO table target
	fmt.Fprintf(&q, `MERGE INTO %s target`, fmtTable)

	// USING (SELECT :1 AS col1, :2 AS col2 FROM DUAL) source
	q.WriteString(` USING (SELECT `)
	for i := range columns {
		if i > 0 {
			q.WriteString(`, `)
		}
		colName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		q.WriteString(formatter.FormatPlaceholder(i))
		q.WriteString(` AS `)
		q.WriteString(colName)
	}
	q.WriteString(` FROM DUAL) source`)

	// ON (target.pk1 = source.pk1 AND target.pk2 = source.pk2)
	q.WriteString(` ON (`)
	for i, col := range conflictCols {
		if i > 0 {
			q.WriteString(` AND `)
		}
		colName, err := formatter.FormatColumnName(col)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&q, `target.%s = source.%s`, colName, colName)
	}
	q.WriteByte(')')

	// WHEN MATCHED THEN UPDATE SET (only if withUpdate)
	if withUpdate {
		q.WriteString(` WHEN MATCHED THEN UPDATE SET`)
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
			colName, err := formatter.FormatColumnName(columns[i].Name)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&q, ` target.%s = source.%s`, colName, colName)
		}
	}

	// WHEN NOT MATCHED THEN INSERT (col1, col2, ...) VALUES (source.col1, source.col2, ...)
	q.WriteString(` WHEN NOT MATCHED THEN INSERT (`)
	for i := range columns {
		if i > 0 {
			q.WriteByte(',')
		}
		colName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		q.WriteString(colName)
	}
	q.WriteString(`) VALUES (`)
	for i := range columns {
		if i > 0 {
			q.WriteByte(',')
		}
		colName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&q, `source.%s`, colName)
	}
	q.WriteByte(')')

	return q.String(), nil
}
