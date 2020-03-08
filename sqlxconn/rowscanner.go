package sqlxconn

import (
	"reflect"

	"github.com/jmoiron/sqlx"

	"github.com/domonda/go-sqldb/impl"
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
		// or pointers to pointers, so set pointer to newly allocated struct
		// and assign result to pointed to pointer in case of success
		if v.Kind() == reflect.Ptr {
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

func (s *rowScanner) ScanStrings() ([]string, error) {
	return impl.ScanStrings(s.row)
}
