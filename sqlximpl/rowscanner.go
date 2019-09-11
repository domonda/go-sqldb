package sqlximpl

import (
	"reflect"

	"github.com/jmoiron/sqlx"
)

type rowScanner struct {
	row *sqlx.Row
}

func (s rowScanner) Scan(dest ...interface{}) error {
	return s.row.Scan(dest...)
}

func (s rowScanner) ScanStruct(dest interface{}) error {
	if v := reflect.ValueOf(dest); v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		// sqlx StructScan does not support pointers to nil pointers
		// so set pointer to newly allocated struct
		if v.Kind() == reflect.Ptr && v.IsNil() {
			n := reflect.New(v.Type().Elem())
			err := s.row.StructScan(n.Interface())
			if err != nil {
				return err
			}
			v.Set(n)
			return nil
		}
	}

	return s.row.StructScan(dest)
}
