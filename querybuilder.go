package sqldb

import (
	"fmt"
	"strings"
)

var DefaultQueryBuilder QueryBuilder = StdQueryBuilder{}

type QueryBuilder interface {
	QueryRowWithPK(formatter QueryFormatter, table string, pkColumns []string) (query string, err error)
	Insert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	InsertUnique(formatter QueryFormatter, table string, columns []ColumnInfo, onConflict string) (query string, err error)
	Upsert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	// Update updates a table rows with the passed values using the
	// passed where clause. That where clause can contain placeholders
	// starting at $1 for the passed whereArgs.
	// It returns queryArgs to be used together with the returned query
	// that combine the passed whereArgs with the passed values.
	Update(formatter QueryFormatter, table string, values Values, where string, whereArgs []any) (query string, queryArgs []any, err error)
	UpdateColumns(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
}

type StdQueryBuilder struct{}

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

func (b StdQueryBuilder) Upsert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
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

func (StdQueryBuilder) UpdateColumns(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
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
