package sqlximpl

import (
	"reflect"

	"github.com/jmoiron/sqlx"

	"github.com/domonda/errors"
	sqldb "github.com/domonda/go-sqldb"
)

type rowsScanner struct {
	rows *sqlx.Rows
}

func (s rowsScanner) ScanSlice(dest interface{}) error {
	defer s.rows.Close()

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return errors.Errorf("ScanSlice needs pointer to slice, got %s", v.Type())
	}
	if v.IsNil() {
		return errors.New("passed nil to ScanSlice")
	}
	slice := v.Elem()
	if v.Kind() != reflect.Slice {
		return errors.Errorf("ScanSlice needs pointer to slice, got pointer to %s", v.Type())
	}

	newLen := 0
	newCap := 16
	newSlice := reflect.MakeSlice(slice.Type(), newLen, newCap)

	for s.rows.Next() {
		if newLen == newCap {
			// Double capacity
			newSlice = reflect.AppendSlice(newSlice, reflect.MakeSlice(slice.Type(), newLen, newLen)).Slice(0, newLen)
			newCap *= 2
		}

		// Grow slice by one
		newLen++
		newSlice = newSlice.Slice(0, newLen)

		lastElemPtr := newSlice.Index(newLen - 1).Addr()
		err := s.rows.Scan(lastElemPtr.Interface())
		if err != nil {
			return err
		}
	}
	if err := s.rows.Err(); err != nil {
		return err
	}

	// Assign newSlice if there were no errors
	slice.Set(newSlice)

	return nil
}

func (s rowsScanner) ScanStructSlice(dest interface{}) error {
	return sqlx.StructScan(s.rows, dest)
}

func (s rowsScanner) ForEachRow(callback func(sqldb.RowScanner) error) error {
	defer s.rows.Close()

	for s.rows.Next() {
		err := callback(rowsWrapper{s.rows})
		if err != nil {
			return err
		}
	}
	return s.rows.Err()
}

type rowsWrapper struct {
	rows *sqlx.Rows
}

func (s rowsWrapper) Scan(dest ...interface{}) error {
	return s.rows.Scan(dest...)
}

func (s rowsWrapper) ScanStruct(dest interface{}) error {
	return s.rows.StructScan(dest)
}
