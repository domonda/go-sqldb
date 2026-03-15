package sqldb

import (
	"fmt"
	"strings"
)

type QueryBuilder interface {
	QueryRowWithPK(formatter QueryFormatter, table string, pkColumns []string) (query string, err error)
	Insert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	// InsertRows builds a multi-row INSERT INTO query with numRows value tuples:
	//   INSERT INTO table(col1,col2) VALUES($1,$2),($3,$4),($5,$6)
	// numRows must be >= 1.
	InsertRows(formatter QueryFormatter, table string, columns []ColumnInfo, numRows int) (query string, err error)
	InsertUnique(formatter QueryFormatter, table string, columns []ColumnInfo, onConflict string) (query string, err error)
	Upsert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	// Update updates a table rows with the passed values using the
	// passed where clause. That where clause can contain placeholders
	// starting at $1 for the passed whereArgs.
	// It returns queryArgs to be used together with the returned query
	// that combine the passed whereArgs with the passed values.
	Update(formatter QueryFormatter, table string, values Values, where string, whereArgs []any) (query string, queryArgs []any, err error)
	UpdateColumns(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	Delete(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
}

// StdQueryBuilder is the default [QueryBuilder] implementation
// that generates standard SQL queries.
type StdQueryBuilder struct{}

// QueryRowWithPK builds a SELECT * query filtered by primary key columns.
func (StdQueryBuilder) QueryRowWithPK(formatter QueryFormatter, table string, pkColumns []string) (query string, err error) {
	var q strings.Builder
	table, err = formatter.FormatTableName(table)
	if err != nil {
		return "", err
	}
	pkCol, err := formatter.FormatColumnName(pkColumns[0])
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&q, `SELECT * FROM %s WHERE %s = %s`, table, pkCol, formatter.FormatPlaceholder(0))
	for i := 1; i < len(pkColumns); i++ {
		pkCol, err = formatter.FormatColumnName(pkColumns[i])
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&q, ` AND %s = %s`, pkCol, formatter.FormatPlaceholder(i))
	}
	return q.String(), nil
}

// Insert builds an INSERT INTO query for the given table and columns.
func (StdQueryBuilder) Insert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
	var q strings.Builder
	table, err = formatter.FormatTableName(table)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&q, `INSERT INTO %s(`, table)
	for i := range columns {
		column := columns[i].Name
		column, err = formatter.FormatColumnName(column)
		if err != nil {
			return "", err
		}
		if i > 0 {
			q.WriteByte(',')
		}
		q.WriteString(column)
	}
	q.WriteString(`) VALUES(`)
	for i := range columns {
		if i > 0 {
			q.WriteByte(',')
		}
		q.WriteString(formatter.FormatPlaceholder(i))
	}
	q.WriteString(`)`)
	return q.String(), nil
}

// InsertRows builds a multi-row INSERT INTO query with numRows value tuples.
// numRows must be >= 1.
func (StdQueryBuilder) InsertRows(formatter QueryFormatter, table string, columns []ColumnInfo, numRows int) (query string, err error) {
	if numRows < 1 {
		return "", fmt.Errorf("InsertRows: numRows must be >= 1, got %d", numRows)
	}
	numCols := len(columns)
	var q strings.Builder
	table, err = formatter.FormatTableName(table)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&q, `INSERT INTO %s(`, table)
	for i := range columns {
		column, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		if i > 0 {
			q.WriteByte(',')
		}
		q.WriteString(column)
	}
	q.WriteString(`) VALUES`)
	for row := range numRows {
		if row > 0 {
			q.WriteByte(',')
		}
		q.WriteByte('(')
		for col := range numCols {
			if col > 0 {
				q.WriteByte(',')
			}
			q.WriteString(formatter.FormatPlaceholder(row*numCols + col))
		}
		q.WriteByte(')')
	}
	return q.String(), nil
}

// InsertUnique builds an INSERT query with ON CONFLICT DO NOTHING.
// The onConflict string specifies the conflict target columns.
func (b StdQueryBuilder) InsertUnique(formatter QueryFormatter, table string, columns []ColumnInfo, onConflict string) (query string, err error) {
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
func (b StdQueryBuilder) Upsert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
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

// Update builds an UPDATE SET ... WHERE query.
// The where clause uses placeholders starting at $1 for whereArgs.
// The returned queryArgs combine whereArgs with the values.
func (StdQueryBuilder) Update(formatter QueryFormatter, table string, values Values, where string, whereArgs []any) (query string, queryArgs []any, err error) {
	var q strings.Builder
	table, err = formatter.FormatTableName(table)
	if err != nil {
		return "", nil, err
	}
	fmt.Fprintf(&q, `UPDATE %s SET`, table)

	columns, vals := values.SortedColumnsAndValues()
	for i := range columns {
		column, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", nil, err
		}
		if i > 0 {
			q.WriteByte(',')
		}
		fmt.Fprintf(&q, ` %s=%s`, column, formatter.FormatPlaceholder(len(whereArgs)+i))
	}
	fmt.Fprintf(&q, ` WHERE %s`, where)

	return q.String(), append(whereArgs, vals...), nil
}

// UpdateColumns builds an UPDATE SET ... WHERE query using column metadata.
// Primary key columns form the WHERE clause, non-primary key columns are SET.
func (StdQueryBuilder) UpdateColumns(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
	hasNonPK := false
	for i := range columns {
		if !columns[i].PrimaryKey {
			hasNonPK = true
			break
		}
	}
	if !hasNonPK {
		return "", fmt.Errorf("UpdateColumns requires at least one non-primary-key column")
	}

	var q strings.Builder
	table, err = formatter.FormatTableName(table)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(&q, `UPDATE %s SET`, table)

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
		fmt.Fprintf(&q, ` %s=%s`, columnName, formatter.FormatPlaceholder(i))
	}

	q.WriteString(` WHERE `)

	first = true
	for i := range columns {
		if !columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			q.WriteString(` AND `)
		}
		columnName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&q, `%s = %s`, columnName, formatter.FormatPlaceholder(i))
	}

	return q.String(), nil
}

// Delete builds a DELETE FROM ... WHERE query using column metadata.
// All provided columns form the WHERE clause.
func (StdQueryBuilder) Delete(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("DeleteColumns requires at least one column")
	}

	var q strings.Builder
	table, err = formatter.FormatTableName(table)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(&q, `DELETE FROM %s WHERE `, table)

	for i := range columns {
		if i > 0 {
			q.WriteString(` AND `)
		}
		columnName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&q, `%s = %s`, columnName, formatter.FormatPlaceholder(i))
	}

	return q.String(), nil
}
