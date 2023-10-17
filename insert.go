package sqldb

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func Insert(ctx context.Context, table string, rows any) error {
	conn := ContextConnection(ctx)

	v := reflect.ValueOf(rows)
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			return nil
		}
		mapped, err := MapStructType(conn, v.Type().Elem())
		if err != nil {
			return err
		}
		return insertRows(ctx, conn, mapped, table, reflect.ValueOf(rows))

	case reflect.Struct:
		columns, values, table, err := MapStructFieldValues(conn, rows)
		if err != nil {
			return err
		}
		query := createInsertQuery(table, columns, 1, conn)
		return conn.Exec(ctx, query, values...)

	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("%T is not a map with a string key type", rows)
		}
		columns, values := mapKeysAndValues(v)
		query := createInsertQuery(table, columns, 1, conn)
		return conn.Exec(ctx, query, values...)

	default:
		return fmt.Errorf("%T not supported as rows argument", rows)
	}
}

func InsertRow(ctx context.Context, row RowWithTableName) error {
	conn := ContextConnection(ctx)
	columns, values, table, err := MapStructFieldValues(conn, row)
	if err != nil {
		return err
	}
	query := createInsertQuery(table, columns, 1, conn)
	return conn.Exec(ctx, query, values...)
}

func InsertRows[R RowWithTableName](ctx context.Context, rows []R) error {
	if len(rows) == 0 {
		return nil
	}
	conn := ContextConnection(ctx)

	mapped, err := MapStructType(conn, reflect.TypeOf(rows[0]))
	if err != nil {
		return err
	}
	return insertRows(ctx, conn, mapped, mapped.Table, reflect.ValueOf(rows))
}

func insertRows(ctx context.Context, conn Connection, mapped *MappedStruct, table string, rows reflect.Value) error {
	numRows := rows.Len()
	maxParams := conn.MaxParameters()
	maxRowsPerInsert := maxParams / len(mapped.Fields)
	if maxRowsPerInsert == 0 {
		return fmt.Errorf("%s has %d mapped struct fields which is greater than Connection.MaxParameters of %d", mapped.Type, len(mapped.Fields), maxParams)
	}

	query := createInsertQuery(mapped.Table, mapped.Columns, maxRowsPerInsert, conn)

	insertValues := make([]any, 0, len(mapped.Fields)*maxRowsPerInsert)
	for i := maxRowsPerInsert; i < numRows; i += maxRowsPerInsert {
		for j := 0; j < maxRowsPerInsert; j++ {
			rowValues, err := mapped.StructFieldValues(rows.Index(i + j))
			if err != nil {
				return err
			}
			insertValues = append(insertValues, rowValues...)
		}

		err := conn.Exec(ctx, query, insertValues...)
		if err != nil {
			return err
		}

		insertValues = insertValues[:0]
	}

	query = createInsertQuery(mapped.Table, mapped.Columns, maxRowsPerInsert, conn)

	// todo last insert

	return nil
}

func createInsertQuery(table string, columns []string, numRows int, formatter QueryFormatter) string {
	var b strings.Builder
	b.WriteString(`INSERT INTO `)
	b.WriteString(table)
	b.WriteByte('(')
	for i, column := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(column)
		b.WriteByte('"')
	}
	b.WriteString(`) VALUES`)
	for r := 0; r < numRows; r++ {
		if r > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		for c := range columns {
			if c > 0 {
				b.WriteByte(',')
			}
			b.WriteString(formatter.ParameterPlaceholder(r*len(columns) + c))
		}
		b.WriteByte(')')
	}
	return b.String()
}
