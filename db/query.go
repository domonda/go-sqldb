package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/domonda/go-sqldb"
)

// Now returns the result of the SQL now()
// function for the current connection.
// Useful for getting the timestamp of a
// SQL transaction for use in Go code.
func Now(ctx context.Context) (time.Time, error) {
	return Conn(ctx).Now()
}

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	return Conn(ctx).Exec(query, args...)
}

// QueryRow queries a single row and returns a RowScanner for the results.
func QueryRow(ctx context.Context, query string, args ...any) sqldb.RowScanner {
	return Conn(ctx).QueryRow(query, args...)
}

// QueryRows queries multiple rows and returns a RowsScanner for the results.
func QueryRows(ctx context.Context, query string, args ...any) sqldb.RowsScanner {
	return Conn(ctx).QueryRows(query, args...)
}

// QueryValue queries a single value of type T.
func QueryValue[T any](ctx context.Context, query string, args ...any) (T, error) {
	var val T
	err := Conn(ctx).QueryRow(query, args...).Scan(&val)
	return val, err
}

// QueryStruct uses the passed pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValues and a table name.
func QueryStruct[S any](ctx context.Context, pkValues ...any) (row *S, err error) {
	if len(pkValues) == 0 {
		return nil, errors.New("missing primary key values")
	}
	t := reflect.TypeOf(row).Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct template type instead of %s", t)
	}
	conn := Conn(ctx)
	table, pkColumns, err := pkColumnsOfStruct(conn, t)
	if err != nil {
		return nil, err
	}
	if len(pkColumns) != len(pkValues) {
		return nil, fmt.Errorf("got %d primary key values, but struct %s has %d primary key fields", len(pkValues), t, len(pkColumns))
	}
	query := fmt.Sprintf(`SELECT * FROM %s WHERE "%s" = $1`, table, pkColumns[0])
	for i := 1; i < len(pkColumns); i++ {
		query += fmt.Sprintf(` AND "%s" = $%d`, pkColumns[i], i+1)
	}
	err = conn.QueryRow(query, pkValues...).ScanStruct(&row)
	if err != nil {
		return nil, err
	}
	return row, nil
}

// QueryStructOrNil uses the passed pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValues.
func QueryStructOrNil[S any](ctx context.Context, pkValues ...any) (row *S, err error) {
	row, err = QueryStruct[S](ctx, pkValues...)
	return row, ReplaceErrNoRows(err, nil)
}

func pkColumnsOfStruct(conn sqldb.Connection, t reflect.Type) (table string, columns []string, err error) {
	mapper := conn.StructFieldMapper()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldTable, column, flags, ok := mapper.MapStructField(field)
		if !ok {
			continue
		}
		if fieldTable != "" && fieldTable != table {
			if table != "" {
				return "", nil, fmt.Errorf("table name not unique (%s vs %s) in struct %s", table, fieldTable, t)
			}
			table = fieldTable
		}

		if column == "" {
			fieldTable, columnsEmbed, err := pkColumnsOfStruct(conn, field.Type)
			if err != nil {
				return "", nil, err
			}
			if fieldTable != "" && fieldTable != table {
				if table != "" {
					return "", nil, fmt.Errorf("table name not unique (%s vs %s) in struct %s", table, fieldTable, t)
				}
				table = fieldTable
			}
			columns = append(columns, columnsEmbed...)
		} else if flags.PrimaryKey() {
			if err = conn.ValidateColumnName(column); err != nil {
				return "", nil, fmt.Errorf("%w in struct field %s.%s", err, t, field.Name)
			}
			columns = append(columns, column)
		}
	}
	return table, columns, nil
}

// QueryStructSlice returns queried rows as slice of pointers to the generic struct type S
func QueryStructSlice[S any](ctx context.Context, query string, args ...any) (rows []*S, err error) {
	err = Conn(ctx).QueryRows(query, args...).ScanStructSlice(&rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
