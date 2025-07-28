package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/domonda/go-sqldb"
)

// UpsertStruct TODO
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, rowStruct sqldb.StructWithTableName, options ...sqldb.QueryOption) error {
	v, err := derefStruct(reflect.ValueOf(rowStruct))
	if err != nil {
		return err
	}
	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(v.Type())
	if err != nil {
		return err
	}
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)
	table, err = conn.FormatTableName(table)
	if err != nil {
		return err
	}

	columns, vals := sqldb.ReflectStructColumnsAndValues(v, reflector, append(options, sqldb.IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col sqldb.ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpsertStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	query, err := queryBuilder.Upsert(table, columns)
	if err != nil {
		return fmt.Errorf("UpsertStruct of table %s: can't create UPSERT query because: %w", table, err)
	}
	return sqldb.Exec(ctx, conn, query, vals...)
}

func UpsertStructStmt[S sqldb.StructWithTableName](ctx context.Context, options ...sqldb.QueryOption) (upsert func(ctx context.Context, rowStruct S) error, done func() error, err error) {
	structType := reflect.TypeFor[S]()
	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	conn := Conn(ctx)
	queryBuilder := QueryBuilderFuncFromContext(ctx)(conn)
	table, err = conn.FormatTableName(table)
	if err != nil {
		return nil, nil, err
	}

	options = append(options, sqldb.IgnoreReadOnly)
	columns := sqldb.ReflectStructColumns(structType, reflector, options...)
	hasPK := slices.ContainsFunc(columns, func(col sqldb.ColumnInfo) bool {
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
		vals := sqldb.ReflectStructValues(v, reflector, options...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return sqldb.WrapErrorWithQuery(err, query, vals, conn)
		}
		return nil
	}
	return upsert, stmt.Close, nil
}

func UpsertStructs[S sqldb.StructWithTableName](ctx context.Context, rowStructs []S, options ...sqldb.QueryOption) error {
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return UpsertStruct(ctx, rowStructs[0], options...)
	}
	return Transaction(ctx, func(ctx context.Context) (e error) {
		upsertFunc, closeFunc, err := UpsertStructStmt[S](ctx, options...)
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
