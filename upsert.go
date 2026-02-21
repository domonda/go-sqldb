package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// UpsertRowStruct inserts a new row or updates an existing one
// if inserting conflicts on the primary key column(s).
// The struct must have at least one field tagged as primary key.
func UpsertRowStruct(ctx context.Context, c *ConnExt, rowStruct StructWithTableName, options ...QueryOption) error {
	v, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}
	table, err := c.StructReflector.TableNameForStruct(v.Type())
	if err != nil {
		return err
	}
	table, err = c.QueryFormatter.FormatTableName(table)
	if err != nil {
		return err
	}

	columns, vals := ReflectStructColumnsAndValues(v, c.StructReflector, append(options, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpsertRowStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	query, err := c.QueryBuilder.Upsert(c.QueryFormatter, table, columns)
	if err != nil {
		return fmt.Errorf("UpsertRowStruct of table %s: can't create UPSERT query because: %w", table, err)
	}
	err = c.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, c.QueryFormatter)
	}
	return nil
}

// UpsertRowStructStmt prepares a statement for upserting rows of type S.
// Returns an upsert function to upsert individual rows and a closeStmt
// function that must be called when done to close the prepared statement.
func UpsertRowStructStmt[S StructWithTableName](ctx context.Context, c *ConnExt, options ...QueryOption) (upsert func(ctx context.Context, rowStruct S) error, closeStmt func() error, err error) {
	structType := reflect.TypeFor[S]()
	table, err := c.StructReflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	table, err = c.QueryFormatter.FormatTableName(table)
	if err != nil {
		return nil, nil, err
	}

	options = append(options, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, c.StructReflector, options...)
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return nil, nil, fmt.Errorf("UpsertRowStructStmt of table %s: %s has no mapped primary key field", table, structType)
	}

	query, err := c.QueryBuilder.Upsert(c.QueryFormatter, table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertRowStructStmt of table %s: can't create UPSERT query because: %w", table, err)
	}

	stmt, err := c.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertRowStructStmt of table %s: can't prepare UPSERT statement because: %w", table, err)
	}

	upsert = func(ctx context.Context, rowStruct S) error {
		v, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals := ReflectStructValues(v, c.StructReflector, options...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, c.QueryFormatter)
		}
		return nil
	}
	return upsert, stmt.Close, nil
}

// UpsertRowStructs upserts a slice of structs within a transaction
// using a prepared statement for efficiency.
func UpsertRowStructs[S StructWithTableName](ctx context.Context, c *ConnExt, rowStructs []S, options ...QueryOption) error {
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return UpsertRowStruct(ctx, c, rowStructs[0], options...)
	}
	return TransactionExt(ctx, c, nil, func(tx *ConnExt) (err error) {
		upsertFunc, closeStmt, stmtErr := UpsertRowStructStmt[S](ctx, tx, options...)
		if stmtErr != nil {
			return stmtErr
		}
		defer func() {
			err = errors.Join(err, closeStmt())
		}()

		for _, rowStruct := range rowStructs {
			err = upsertFunc(ctx, rowStruct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
