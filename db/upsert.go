package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/domonda/go-sqldb"
)

// UpsertStruct TODO
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, rowStruct StructWithTableName, options ...QueryOption) error {
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
	table, err = conn.FormatTableName(table)
	if err != nil {
		return err
	}

	columns, vals := ReflectStructColumnsAndValues(v, reflector, append(options, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col sqldb.ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpsertStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	queryBuilder := QueryBuilderFromContext(ctx)

	var query strings.Builder
	err = queryBuilder.Upsert(&query, table, columns)
	if err != nil {
		return fmt.Errorf("UpsertStruct of table %s: can't create UPSERT query because: %w", table, err)
	}
	return sqldb.Exec(ctx, conn, query.String(), vals...)
}

func UpsertStructStmt[S StructWithTableName](ctx context.Context, options ...QueryOption) (upsert func(ctx context.Context, rowStruct S) error, done func() error, err error) {
	structType := reflect.TypeFor[S]()
	reflector := GetStructReflector(ctx)
	table, err := reflector.TableNameForStruct(structType)
	if err != nil {
		return nil, nil, err
	}
	conn := Conn(ctx)
	table, err = conn.FormatTableName(table)
	if err != nil {
		return nil, nil, err
	}

	options = append(options, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, reflector, options...)
	hasPK := slices.ContainsFunc(columns, func(col sqldb.ColumnInfo) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: %s has no mapped primary key field", table, structType)
	}

	queryBuilder := QueryBuilderFromContext(ctx)

	var query strings.Builder
	err = queryBuilder.Upsert(&query, table, columns)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: can't create UPSERT query because: %w", table, err)
	}

	stmt, err := conn.Prepare(ctx, query.String())
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
			return sqldb.WrapErrorWithQuery(err, query.String(), vals, conn)
		}
		return nil
	}
	return upsert, stmt.Close, nil
}

func UpsertStructs[S StructWithTableName](ctx context.Context, rowStructs []S, options ...QueryOption) error {
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
