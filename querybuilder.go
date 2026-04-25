package sqldb

import (
	"fmt"
	"strings"
)

// QueryBuilder builds standard SQL queries for common CRUD operations.
// For driver-specific operations, use type assertions to check
// for [UpsertQueryBuilder] and [ReturningQueryBuilder] support.
type QueryBuilder interface {
	QueryRowWithPK(formatter QueryFormatter, table string, pkColumns []string) (query string, err error)
	Insert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	// InsertRows builds a multi-row INSERT INTO query with numRows value tuples,
	// using placeholders emitted by the formatter (PostgreSQL syntax shown):
	//   INSERT INTO table(col1,col2) VALUES($1,$2),($3,$4),($5,$6)
	// numRows must be >= 1.
	InsertRows(formatter QueryFormatter, table string, columns []ColumnInfo, numRows int) (query string, err error)
	// Update updates table rows with the passed values using the
	// passed whereCondition.
	//
	// whereCondition is the boolean expression that follows the WHERE
	// keyword. It must NOT include the WHERE keyword itself, which the
	// builder emits. The expression can reference table columns,
	// comparison and logical operators (=, <>, <, IN, IS NULL, AND, OR,
	// NOT, ...), parentheses, and placeholders numbered starting at $1
	// (or the driver's equivalent placeholder syntax) bound to whereArgs
	// in order. Examples: "id = $1", "tenant_id = $1 AND status IN ($2, $3)".
	//
	// It returns queryArgs to be used together with the returned query
	// that combine the passed whereArgs with the passed values.
	//
	// SECURITY: whereCondition is concatenated into the generated SQL
	// verbatim and is NOT parameterized or validated. It must be static
	// SQL written by the developer. Never include values that originated
	// from external input (HTTP requests, filenames, externally populated
	// database content, etc.); pass those through whereArgs using the
	// driver's placeholder syntax.
	Update(formatter QueryFormatter, table string, values Values, whereCondition string, whereArgs []any) (query string, queryArgs []any, err error)
	UpdateColumns(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	Delete(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
}

// UpsertQueryBuilder builds driver-specific upsert and insert-unique queries.
// Not all databases support these operations with the same syntax:
//   - PostgreSQL/SQLite: ON CONFLICT ... DO UPDATE SET / DO NOTHING
//   - MySQL: ON DUPLICATE KEY UPDATE
//   - MSSQL: MERGE
//   - Oracle: MERGE INTO ... USING (SELECT ... FROM DUAL)
//
// Use a type assertion from [QueryBuilder] to check for support:
//
//	uqb, ok := builder.(UpsertQueryBuilder)
//
// conflictTarget is a comma-separated list of column names that
// identify the uniqueness target. The name keeps PostgreSQL terminology
// (ON CONFLICT) for historical reasons, but each driver translates the
// columns into its own vendor syntax. PostgreSQL and SQLite emit
// `ON CONFLICT (cols) DO NOTHING`; MySQL emits
// `ON DUPLICATE KEY UPDATE col = col` using only the first column;
// MSSQL and Oracle emit a `MERGE INTO ... ON target.col = source.col`
// using all columns as merge keys. In every implementation the argument
// is just the column list. It must NOT include the `ON CONFLICT`,
// `ON DUPLICATE KEY UPDATE`, `MERGE`, or any other keyword: the builder
// emits the surrounding clause itself.
//
// SECURITY: the PostgreSQL and SQLite builders embed conflictTarget
// verbatim (after stripping a single pair of outer parentheses), so it
// must be a static, developer-written column list. The MySQL, MSSQL and
// Oracle builders split it on commas and validate each name via the
// formatter, but passing external input is still discouraged as a
// defense-in-depth measure.
type UpsertQueryBuilder interface {
	Upsert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error)
	InsertUnique(formatter QueryFormatter, table string, columns []ColumnInfo, conflictTarget string) (query string, err error)
}

// ReturningQueryBuilder builds queries that return result rows
// using driver-specific syntax (e.g. PostgreSQL/SQLite RETURNING clause).
// Not all databases support RETURNING; use a type assertion
// from [QueryBuilder] to check for support:
//
//	rqb, ok := builder.(ReturningQueryBuilder)
//
// returningColumns is the column or expression list that follows the
// RETURNING keyword and must NOT include the keyword itself.
//
// whereCondition is the boolean expression that follows the WHERE keyword
// and must NOT include the keyword itself. It can reference columns,
// operators, parentheses, and placeholders bound to whereArgs in order.
//
// SECURITY: both returningColumns and whereCondition are concatenated into
// the generated SQL verbatim and are NOT parameterized. They must be
// static SQL written by the developer and must not contain data from
// external input. Pass external input through whereArgs using the driver's
// placeholder syntax.
type ReturningQueryBuilder interface {
	InsertReturning(formatter QueryFormatter, table string, columns []ColumnInfo, returningColumns string) (query string, err error)
	UpdateReturning(formatter QueryFormatter, table string, values Values, returningColumns, whereCondition string, whereArgs []any) (query string, queryArgs []any, err error)
}

// StdQueryBuilder implements [QueryBuilder]
// using standard SQL for CRUD operations.
// It does not implement [UpsertQueryBuilder] or [ReturningQueryBuilder];
// those are provided by driver-specific builders
// (e.g. pqconn.QueryBuilder, mysqlconn.QueryBuilder, mssqlconn.QueryBuilder).
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

// Update builds an UPDATE SET ... WHERE query.
//
// whereCondition is the boolean expression that follows the WHERE keyword.
// It must NOT include the WHERE keyword itself, which the builder emits.
// The expression can reference table columns, comparison and logical
// operators (=, <>, <, IN, IS NULL, AND, OR, NOT, ...), parentheses,
// and placeholders numbered starting at $1 (or the driver's equivalent
// placeholder syntax) bound to whereArgs in order. Examples:
//
//	"id = $1"
//	"tenant_id = $1 AND status IN ($2, $3)"
//	"created_at < $1 AND deleted_at IS NULL"
//
// The returned queryArgs combine whereArgs (bound to the placeholders in
// whereCondition) with the values from the values map (bound to the SET
// placeholders that follow whereArgs).
//
// SECURITY: whereCondition is concatenated into the generated SQL verbatim
// and is NOT parameterized or validated. It must be static SQL written by
// the developer. Never include data that originated from external input
// (HTTP requests, filenames, externally populated database content, etc.);
// pass such data through whereArgs using placeholders.
func (StdQueryBuilder) Update(formatter QueryFormatter, table string, values Values, whereCondition string, whereArgs []any) (query string, queryArgs []any, err error) {
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
	fmt.Fprintf(&q, ` WHERE %s`, whereCondition)

	return q.String(), append(whereArgs, vals...), nil
}

// UpdateColumns builds an UPDATE SET ... WHERE query using column metadata.
// Primary key columns form the WHERE clause, non-primary key columns are SET.
// Placeholders are numbered sequentially: non-PK columns first (SET), then PK columns (WHERE).
// Callers must provide values in the same order (non-PK first, then PK).
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

	placeholder := 0
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
		fmt.Fprintf(&q, ` %s=%s`, columnName, formatter.FormatPlaceholder(placeholder))
		placeholder++
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
		fmt.Fprintf(&q, `%s = %s`, columnName, formatter.FormatPlaceholder(placeholder))
		placeholder++
	}

	return q.String(), nil
}

// Delete builds a DELETE FROM ... WHERE query using column metadata.
// All provided columns form the WHERE clause.
func (StdQueryBuilder) Delete(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("Delete requires at least one column")
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

// StdReturningQueryBuilder extends [StdQueryBuilder] with
// PostgreSQL/SQLite-compatible RETURNING clause support.
// It implements [QueryBuilder] and [ReturningQueryBuilder].
type StdReturningQueryBuilder struct {
	StdQueryBuilder
}

// InsertReturning builds an INSERT INTO query with a RETURNING clause.
//
// returningColumns is the column or expression list that follows the
// RETURNING keyword and must NOT include the keyword itself.
//
// SECURITY: returningColumns is appended to the query verbatim. It must be
// static SQL written by the developer and must not contain data from
// external input.
func (b StdReturningQueryBuilder) InsertReturning(formatter QueryFormatter, table string, columns []ColumnInfo, returningColumns string) (query string, err error) {
	query, err = b.Insert(formatter, table, columns)
	if err != nil {
		return "", err
	}
	return query + " RETURNING " + returningColumns, nil
}

// UpdateReturning builds an UPDATE SET ... WHERE query with a RETURNING clause.
//
// returningColumns is the column or expression list following the RETURNING
// keyword (without the keyword). whereCondition is the boolean expression
// following WHERE (without the keyword) and may contain placeholders bound
// to whereArgs.
//
// SECURITY: both returningColumns and whereCondition are concatenated into
// the SQL verbatim and are NOT parameterized. They must be static SQL
// written by the developer. Pass external input only through whereArgs
// using the driver's placeholder syntax.
func (b StdReturningQueryBuilder) UpdateReturning(formatter QueryFormatter, table string, values Values, returningColumns, whereCondition string, whereArgs []any) (query string, queryArgs []any, err error) {
	query, queryArgs, err = b.Update(formatter, table, values, whereCondition, whereArgs)
	if err != nil {
		return "", nil, err
	}
	return query + " RETURNING " + returningColumns, queryArgs, nil
}
