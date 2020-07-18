package sqlxconn

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// rowsScanner implements sqldb.RowsScanner for sqlx.Rows
type rowsScanner struct {
	ctx   context.Context
	query string // for error wrapping
	rows  *sqlx.Rows
}

func (s *rowsScanner) scanSlice(dest interface{}, scanStruct bool) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query `%s` returned error: %w", s.query, err)
		}
		s.rows.Close()
	}()

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("scan dest is not a pointer but %s", destVal.Type())
	}
	if destVal.IsNil() {
		return errors.New("scan dest is nil")
	}
	slice := destVal.Elem()
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("scan dest is not pointer to slice but %s", destVal.Type())
	}
	sliceElemType := slice.Type().Elem()

	newSlice := reflect.MakeSlice(slice.Type(), 0, 32)

	for s.rows.Next() {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}

		newSlice = reflect.Append(newSlice, reflect.Zero(sliceElemType))
		target := newSlice.Index(newSlice.Len() - 1)
		if sliceElemType.Kind() == reflect.Ptr {
			// sqlx does not allocate for pointer types,
			// so set last slice element to newly allocated object
			target.Set(reflect.New(sliceElemType.Elem()))
		} else {
			// If no pointer type, then use address of last slice element
			target = target.Addr()
		}
		if scanStruct {
			err = s.rows.StructScan(target.Interface())
		} else {
			err = s.rows.Scan(target.Interface())
		}
		if err != nil {
			return err
		}
	}
	if s.rows.Err() != nil {
		return s.rows.Err()
	}

	// Assign newSlice if there were no errors
	if newSlice.Len() == 0 {
		slice.SetLen(0)
	} else {
		slice.Set(newSlice)
	}

	return nil
}

func (s *rowsScanner) ScanSlice(dest interface{}) error {
	return s.scanSlice(dest, false)
}

func (s *rowsScanner) ScanStructSlice(dest interface{}) error {
	return s.scanSlice(dest, true)
}

func (s *rowsScanner) ScanStrings(headerRow bool) (rows [][]string, err error) {
	if headerRow {
		columns, err := s.rows.Columns()
		if err != nil {
			return nil, err
		}
		rows = [][]string{columns}
	}
	err = s.ForEachRow(func(rowScanner sqldb.RowScanner) error {
		row, err := rowScanner.ScanStrings()
		if err != nil {
			return err
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *rowsScanner) ForEachRow(callback func(sqldb.RowScanner) error) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query `%s` returned error: %w", s.query, err)
		}
		s.rows.Close()
	}()

	for s.rows.Next() {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}

		err := callback(perRowScanner{s.rows})
		if err != nil {
			return err
		}
	}
	return s.rows.Err()
}

func (s *rowsScanner) ForEachRowScan(callback interface{}) error {
	forEachRowFunc, err := impl.ForEachRowScanFunc(s.ctx, callback)
	if err != nil {
		return err
	}
	return s.ForEachRow(forEachRowFunc)
}

type perRowScanner struct {
	rows *sqlx.Rows
}

func (s perRowScanner) Scan(dest ...interface{}) error {
	return s.rows.Scan(dest...)
}

func (s perRowScanner) ScanStruct(dest interface{}) error {
	return s.rows.StructScan(dest)
}

func (s perRowScanner) ScanStrings() ([]string, error) {
	return impl.ScanStrings(s.rows)
}
