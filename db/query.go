package db

import (
	"context"
	"database/sql"
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
	var now time.Time
	err := Conn(ctx).QueryRow(ctx, `SELECT now()`).Scan(&now)
	if err != nil {
		return time.Time{}, err
	}
	return now, nil
}

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...any) error {
	return Conn(ctx).Exec(ctx, query, args...)
}

// QueryRow queries a single row and returns a Row for the results.
func QueryRow(ctx context.Context, query string, args ...any) sqldb.Row {
	return Conn(ctx).QueryRow(ctx, query, args...)
}

// QueryRows queries multiple rows and returns a Rows for the results.
func QueryRows(ctx context.Context, query string, args ...any) sqldb.Rows {
	return Conn(ctx).QueryRows(ctx, query, args...)
}

// QueryValue queries a single value of type T.
func QueryValue[T any](ctx context.Context, query string, args ...any) (value T, err error) {
	err = Conn(ctx).QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}

// QueryValueOrDefault queries a single value of type T
// or returns the default zero value of T in case of sql.ErrNoRows.
func QueryValueOrDefault[T any](ctx context.Context, query string, args ...any) (value T, err error) {
	err = Conn(ctx).QueryRow(ctx, query, args...).Scan(&value)
	if err != nil {
		var zero T
		if errors.Is(err, sql.ErrNoRows) {
			return zero, nil
		}
		return zero, err
	}
	return value, err
}

// QueryStruct uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
func QueryStruct[S any](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	// Using explicit first pkValue value
	// to not be able to compile without any value
	pkValues = append([]any{pkValue}, pkValues...)
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
	err = conn.QueryRow(ctx, query, pkValues...).ScanStruct(&row)
	if err != nil {
		return nil, err
	}
	return row, nil
}

// QueryStructOrNil uses the passed pkValue+pkValues to query a table row
// and scan it into a struct of type S that must have tagged fields
// with primary key flags to identify the primary key column names
// for the passed pkValue+pkValues and a table name.
// Returns nil as row and error if no row could be found with the
// passed pkValue+pkValues.
func QueryStructOrNil[S any](ctx context.Context, pkValue any, pkValues ...any) (row *S, err error) {
	row, err = QueryStruct[S](ctx, pkValue, pkValues...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
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

// QueryStructSlice returns queried rows as slice of the generic type S
// which must be a struct or a pointer to a struct.
func QueryStructSlice[S any](ctx context.Context, query string, args ...any) (rows []S, err error) {
	err = Conn(ctx).QueryRows(ctx, query, args...).ScanStructSlice(&rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
