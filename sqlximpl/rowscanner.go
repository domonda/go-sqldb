package sqlximpl

import (
	"reflect"

	"github.com/jmoiron/sqlx"

	"github.com/domonda/go-wraperr"
)

// rowScanner implements sqldb.RowScanner for a sqlx.Row
type rowScanner struct {
	query string // for error wrapping
	row   *sqlx.Row
}

func (s *rowScanner) Scan(dest ...interface{}) error {
	err := s.row.Scan(dest...)
	if err != nil {
		return wraperr.Errorf("query `%s` returned error: %w", s.query, err)
	}
	return nil
}

func (s *rowScanner) ScanStruct(dest interface{}) (err error) {
	defer func() {
		if err != nil {
			err = wraperr.Errorf("query `%s` returned error: %w", s.query, err)
		}
	}()

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
