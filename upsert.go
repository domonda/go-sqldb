package sqldb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// UpsertStruct TODO
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, conn Executor, queryBuilder QueryBuilder, reflector StructReflector, rowStruct StructWithTableName, options ...QueryOption) error {
	v, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}
	table, err := reflector.TableNameForStruct(v.Type())
	if err != nil {
		return err
	}
	table, err = queryBuilder.FormatTableName(table)
	if err != nil {
		return err
	}

	columns, vals := ReflectStructColumnsAndValues(v, reflector, append(options, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpsertStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	query, err := queryBuilder.Upsert(table, columns)
	if err != nil {
		return fmt.Errorf("UpsertStruct of table %s: can't create UPSERT query because: %w", table, err)
	}
	err = conn.Exec(ctx, query, vals...)
	if err != nil {
		return WrapErrorWithQuery(err, query, vals, queryBuilder)
	}
	return nil
}

func UpsertStructStmt[S StructWithTableName](ctx context.Context, conn Preparer, queryBuilder QueryBuilder, reflector StructReflector, options ...QueryOption) (upsert func(ctx context.Context, rowStruct S) error, done func() error, err error) {
	structType := reflect.TypeFor[S]()
	table, err := reflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	table, err = queryBuilder.FormatTableName(table)
	if err != nil {
		return nil, nil, err
	}

	options = append(options, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, reflector, options...)
	hasPK := slices.ContainsFunc(columns, func(col ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: %s has no mapped primary key field", table, structType)
	}

	query, err := queryBuilder.Upsert(table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: can't create UPSERT query because: %w", table, err)
	}

	stmt, err := conn.Prepare(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: can't prepare UPSERT statement because: %w", table, err)
	}

	upsert = func(ctx context.Context, rowStruct S) error {
		v, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals := ReflectStructValues(v, reflector, options...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return WrapErrorWithQuery(err, query, vals, queryBuilder)
		}
		return nil
	}
	return upsert, stmt.Close, nil
}

func UpsertStructs[S StructWithTableName](ctx context.Context, conn Connection, queryBuilder QueryBuilder, reflector StructReflector, rowStructs []S, options ...QueryOption) error {
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return UpsertStruct(ctx, conn, queryBuilder, reflector, rowStructs[0], options...)
	}
	return Transaction(ctx, conn, nil, func(tx Connection) (e error) {
		upsertFunc, closeFunc, err := UpsertStructStmt[S](ctx, tx, queryBuilder, reflector, options...)
		if err != nil {
			return err
		}
		defer func() {
			e = errors.Join(e, closeFunc())
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
