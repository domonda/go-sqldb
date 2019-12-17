package sqlximpl

import (
	"reflect"

	"github.com/jmoiron/sqlx"

	"github.com/domonda/errors"
	sqldb "github.com/domonda/go-sqldb"
)

// rowsScanner implements sqldb.RowsScanner for sqlx.Rows
type rowsScanner struct {
	rows *sqlx.Rows
}

func (s rowsScanner) scanSlice(dest interface{}, scanStruct bool) (err error) {
	defer s.rows.Close()

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return errors.Errorf("scan dest is not a pointer but %s", destVal.Type())
	}
	if destVal.IsNil() {
		return errors.New("scan dest is nil")
	}
	slice := destVal.Elem()
	if slice.Kind() != reflect.Slice {
		return errors.Errorf("scan dest is not pointer to slice but %s", destVal.Type())
	}
	sliceElemType := slice.Type().Elem()

	newSlice := reflect.MakeSlice(slice.Type(), 0, 32)

	for s.rows.Next() {
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
	if err := s.rows.Err(); err != nil {
		return err
	}

	// Assign newSlice if there were no errors
	if newSlice.Len() == 0 {
		slice.SetLen(0)
	} else {
		slice.Set(newSlice)
	}

	return nil
}

func (s rowsScanner) ScanSlice(dest interface{}) error {
	return s.scanSlice(dest, false)
}

func (s rowsScanner) ScanStructSlice(dest interface{}) error {
	return s.scanSlice(dest, true)
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
