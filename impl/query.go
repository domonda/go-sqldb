package impl

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/domonda/go-sqldb"
)

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func Exec(ctx context.Context, conn Execer, query string, args []any, converter driver.ValueConverter, argFmt string) error {
	err := convertValuesInPlace(args, converter)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return err
	}
	_, err = conn.ExecContext(ctx, query, args...)
	return WrapNonNilErrorWithQuery(err, query, argFmt, args)
}

type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func QueryRow(ctx context.Context, conn Queryer, query string, args []any, converter driver.ValueConverter, argFmt string, mapper sqldb.StructFieldMapper) sqldb.RowScanner {
	err := convertValuesInPlace(args, converter)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowScannerWithError(err)
	}
	return NewRowScanner(rows, mapper, query, argFmt, args)
}

func QueryRows(ctx context.Context, conn Queryer, query string, args []any, converter driver.ValueConverter, argFmt string, mapper sqldb.StructFieldMapper) sqldb.RowsScanner {
	err := convertValuesInPlace(args, converter)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		err = WrapNonNilErrorWithQuery(err, query, argFmt, args)
		return sqldb.RowsScannerWithError(err)
	}
	return NewRowsScanner(ctx, rows, mapper, query, argFmt, args)
}
