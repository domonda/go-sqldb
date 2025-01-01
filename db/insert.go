package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/domonda/go-sqldb"
)

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values Values) error {
	if len(values) == 0 {
		return fmt.Errorf("Insert into table %s: no values", table)
	}

	conn := Conn(ctx)
	query := strings.Builder{}
	cols, vals := values.SortedColumnsAndValues()

	err := writeInsertQuery(&query, table, cols, conn)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}
	err = conn.Exec(ctx, query.String(), vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return nil
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values Values, onConflict string) (inserted bool, err error) {
	if len(values) == 0 {
		return false, fmt.Errorf("InsertUnique into table %s: no values", table)
	}
	conn := Conn(ctx)

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	var query strings.Builder
	cols, vals := values.SortedColumnsAndValues()
	err = writeInsertQuery(&query, table, cols, conn)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}
	fmt.Fprintf(&query, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)

	inserted, err = QueryValue[bool](ctx, query.String(), vals...)
	err = sqldb.ReplaceErrNoRows(err, nil)
	if err != nil {
		return false, wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return inserted, err
}

// // InsertReturning inserts a new row into table using values
// // and returns values from the inserted row listed in returning.
// func InsertReturning(ctx context.Context, table string, values Values, returning string) sqldb.RowScanner {
// 	if len(values) == 0 {
// 		return sqldb.RowScannerWithError(fmt.Errorf("InsertReturning into table %s: no values", table))
// 	}
// 	conn := Conn(ctx)

// 	var query strings.Builder
// 	names, vals := values.Sorted()
// 	err = writeInsertQuery(&query, table, names, conn)
// 	query.WriteString(" RETURNING ")
// 	query.WriteString(returning)
// 	return conn.QueryRow(query.String(), vals...) // TODO wrap error with query
// }

// InsertStruct inserts a new row into table.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStruct(ctx context.Context, rowStruct StructWithTableName, ignoreColumns ...ColumnFilter) error {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}

	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(structVal.Type())
	if err != nil {
		return err
	}

	columns, vals := ReflectStructColumnsAndValues(structVal, reflector, append(ignoreColumns, IgnoreReadOnly)...)
	conn := Conn(ctx)

	query := strings.Builder{}
	err = writeInsertQuery(&query, table, columns, conn)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}

	err = conn.Exec(ctx, query.String(), vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return nil
}

func InsertStructStmt[S StructWithTableName](ctx context.Context, ignoreColumns ...ColumnFilter) (insertFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
	reflector := GetStructReflector(ctx)
	structType := reflect.TypeFor[S]()
	table, err := reflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	conn := Conn(ctx)
	ignoreColumns = append(ignoreColumns, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, reflector, ignoreColumns...)
	query := strings.Builder{}
	err = writeInsertQuery(&query, table, columns, conn)
	if err != nil {
		return nil, nil, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	stmt, err := conn.Prepare(ctx, query.String())
	if err != nil {
		return nil, nil, fmt.Errorf("can't prepare INSERT query because: %w", err)
	}

	insertFunc = func(ctx context.Context, rowStruct S) error {
		strct, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals := ReflectStructValues(strct, reflector, ignoreColumns...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return wrapErrorWithQuery(err, query.String(), vals, conn)
		}
		return nil
	}
	return insertFunc, stmt.Close, nil
}

// func InsertStructStmt[S StructWithTableName](ctx context.Context, query string) (stmtFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
// 	conn := Conn(ctx)
// 	stmt, err := conn.Prepare(ctx, query)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	stmtFunc = func(ctx context.Context, rowStruct S) error {
// 		TODO
// 		if err != nil {
// 			return wrapErrorWithQuery(err, query, args, conn)
// 		}
// 		return nil
// 	}
// 	return stmtFunc, stmt.Close, nil
// }

// InsertUniqueStruct inserts a new row with unique private key.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns true if a row was inserted.
func InsertUniqueStruct(ctx context.Context, rowStruct StructWithTableName, onConflict string, ignoreColumns ...ColumnFilter) (inserted bool, err error) {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return false, err
	}

	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(structVal.Type())
	if err != nil {
		return false, err
	}

	columns, vals := ReflectStructColumnsAndValues(structVal, reflector, append(ignoreColumns, IgnoreReadOnly)...)
	conn := Conn(ctx)

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	var query strings.Builder
	err = writeInsertQuery(&query, table, columns, conn)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}
	fmt.Fprintf(&query, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)

	inserted, err = QueryValue[bool](ctx, query.String(), vals...)
	err = sqldb.ReplaceErrNoRows(err, nil)
	if err != nil {
		return false, wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return inserted, err
}

// InsertStructs inserts a slice structs
// as new rows into table using the DefaultStructReflector.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertStructs[S StructWithTableName](ctx context.Context, rowStructs []S, ignoreColumns ...ColumnFilter) error {
	// TODO optimized version that combines multiple structs in one query depending or maxArgs
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return InsertStruct(ctx, rowStructs[0], ignoreColumns...)
	}
	return Transaction(ctx, func(ctx context.Context) (e error) {
		insertFunc, closeFunc, err := InsertStructStmt[S](ctx, ignoreColumns...)
		if err != nil {
			return err
		}
		defer func() {
			e = errors.Join(e, closeFunc())
		}()

		for _, rowStruct := range rowStructs {
			err = insertFunc(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func writeInsertQuery(w *strings.Builder, table string, columns []Column, f sqldb.QueryFormatter) (err error) {
	table, err = f.FormatTableName(table)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, `INSERT INTO %s(`, table)
	for i := range columns {
		column := columns[i].Name
		column, err = f.FormatColumnName(column)
		if err != nil {
			return err
		}
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteString(column)
	}
	w.WriteString(`) VALUES(`)
	for i := range columns {
		if i > 0 {
			w.WriteByte(',')
		}
		w.WriteString(f.FormatPlaceholder(i))
	}
	w.WriteByte(')')
	return nil
}
