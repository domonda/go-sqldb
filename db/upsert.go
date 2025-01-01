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
func UpsertStruct(ctx context.Context, rowStruct StructWithTableName, ignoreColumns ...ColumnFilter) error {
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

	columns, vals := ReflectStructColumnsAndValues(v, reflector, append(ignoreColumns, IgnoreReadOnly)...)
	hasPK := slices.ContainsFunc(columns, func(col Column) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return fmt.Errorf("UpsertStruct of table %s: %s has no mapped primary key field", table, v.Type())
	}

	var query strings.Builder
	err = buildUpsertQuery(&query, table, columns, conn)
	if err != nil {
		return fmt.Errorf("UpsertStruct of table %s: can't create UPSERT query because: %w", table, err)
	}
	err = conn.Exec(ctx, query.String(), vals...)
	if err != nil {
		return wrapErrorWithQuery(err, query.String(), vals, conn)
	}
	return nil
}

func UpsertStructStmt[S StructWithTableName](ctx context.Context, ignoreColumns ...ColumnFilter) (upsertFunc func(ctx context.Context, rowStruct S) error, closeFunc func() error, err error) {
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

	ignoreColumns = append(ignoreColumns, IgnoreReadOnly)
	columns := ReflectStructColumns(structType, reflector, ignoreColumns...)
	hasPK := slices.ContainsFunc(columns, func(col Column) bool {
		return col.PrimaryKey
	})
	if !hasPK {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: %s has no mapped primary key field", table, structType)
	}

	var query strings.Builder
	err = buildUpsertQuery(&query, table, columns, conn)
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: can't create UPSERT query because: %w", table, err)
	}

	stmt, err := conn.Prepare(ctx, query.String())
	if err != nil {
		return nil, nil, fmt.Errorf("UpsertStructStmt of table %s: can't prepare UPSERT statement because: %w", table, err)
	}

	upsertFunc = func(ctx context.Context, rowStruct S) error {
		v, err := derefStruct(reflect.ValueOf(rowStruct))
		if err != nil {
			return err
		}
		vals := ReflectStructValues(v, reflector, ignoreColumns...)
		err = stmt.Exec(ctx, vals...)
		if err != nil {
			return wrapErrorWithQuery(err, query.String(), vals, conn)
		}
		return nil
	}
	return upsertFunc, stmt.Close, nil
}

func UpsertStructs[S StructWithTableName](ctx context.Context, rowStructs []S, ignoreColumns ...ColumnFilter) error {
	switch len(rowStructs) {
	case 0:
		return nil
	case 1:
		return UpsertStruct(ctx, rowStructs[0], ignoreColumns...)
	}
	return Transaction(ctx, func(ctx context.Context) (e error) {
		upsertFunc, closeFunc, err := UpsertStructStmt[S](ctx, ignoreColumns...)
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

func buildUpsertQuery(w *strings.Builder, table string, columns []Column, f sqldb.QueryFormatter) (err error) {
	err = buildInsertQuery(w, table, columns, f)
	if err != nil {
		return err
	}
	w.WriteString(` ON CONFLICT(`)
	first := true
	for i := range columns {
		if !columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			w.WriteByte(',')
		}
		columnName, err := f.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		w.WriteString(columnName)
	}
	w.WriteString(`) DO UPDATE SET`)
	first = true
	for i := range columns {
		if columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			w.WriteByte(',')
		}
		columnName, err := f.FormatColumnName(columns[i].Name)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, ` %s=%s`, columnName, f.FormatPlaceholder(i))
	}
	return nil
}
