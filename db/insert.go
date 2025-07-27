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
func Insert(ctx context.Context, table string, values sqldb.Values) error {
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)
	return sqldb.Insert(ctx, conn, queryBuilder, table, values)
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)
	return sqldb.InsertUnique(ctx, conn, queryBuilder, table, values, onConflict)
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
// 	err = writeInsert(&query, table, names, conn)
// 	query.WriteString(" RETURNING ")
// 	query.WriteString(returning)
// 	return conn.QueryRow(query.String(), vals...) // TODO wrap error with query
// }

// InsertRowStruct inserts a new row into table.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStruct(ctx context.Context, rowStruct StructWithTableName, options ...QueryOption) error {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}

	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(structVal.Type())
	if err != nil {
		return err
	}

	columns, vals := ReflectStructColumnsAndValues(structVal, reflector, append(options, IgnoreReadOnly)...)
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)

	query, err := queryBuilder.Insert(table, columns)
	if err != nil {
		return fmt.Errorf("can't create INSERT query because: %w", err)
	}

	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return sqldb.WrapErrorWithQuery(err, query, vals, conn)
	}
	return nil
}

func InsertRowStructStmt[S StructWithTableName](ctx context.Context, options ...QueryOption) (insertFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
	reflector := GetStructReflector(ctx)
	structType := reflect.TypeFor[S]()
	table, err := reflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)
	options = append(options, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, reflector, options...)

	query, err := queryBuilder.Insert(table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("can't prepare INSERT query because: %w", err)
	}

	insertFunc = func(ctx context.Context, rowStruct S) error {
		strct, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals := ReflectStructValues(strct, reflector, options...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return sqldb.WrapErrorWithQuery(err, query, vals, conn)
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
// 			return sqldb.WrapErrorWithQuery(err, query, args, conn)
// 		}
// 		return nil
// 	}
// 	return stmtFunc, stmt.Close, nil
// }

// InsertUniqueRowStruct inserts a new row with unique private key.
// Optional ColumnFilter can be passed to ignore mapped columns.
// Does nothing if the onConflict statement applies
// and returns true if a row was inserted.
func InsertUniqueRowStruct(ctx context.Context, rowStruct StructWithTableName, onConflict string, options ...QueryOption) (inserted bool, err error) {
	structVal, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return false, err
	}

	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(structVal.Type())
	if err != nil {
		return false, err
	}

	columns, vals := ReflectStructColumnsAndValues(structVal, reflector, append(options, IgnoreReadOnly)...)
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)

	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}

	query, err := queryBuilder.InsertUnique(table, columns, onConflict)
	if err != nil {
		return false, fmt.Errorf("can't create INSERT query because: %w", err)
	}

	inserted, err = QueryRowValue[bool](ctx, query, vals...)
	err = sqldb.ReplaceErrNoRows(err, nil)
	if err != nil {
		return false, sqldb.WrapErrorWithQuery(err, query, vals, conn)
	}
	return inserted, err
}

// InsertRowStructs inserts a slice structs
// as new rows into table using the DefaultStructReflector.
// Optional ColumnFilter can be passed to ignore mapped columns.
func InsertRowStructs[S StructWithTableName](ctx context.Context, rowStructs []S, options ...QueryOption) error {
	// TODO optimized version that combines multiple structs in one query depending or maxArgs
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return InsertRowStruct(ctx, rowStructs[0], options...)
	}
	return Transaction(ctx, func(ctx context.Context) (e error) {
		insert, done, err := InsertRowStructStmt[S](ctx, options...)
		if err != nil {
			return err
		}
		defer func() {
			e = errors.Join(e, done())
		}()

		for _, rowStruct := range rowStructs {
			err = insert(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
