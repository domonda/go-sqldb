package pqtimpl

import (
	"database/sql"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
	"github.com/domonda/go-wraperr"
)

// rowScanner implements sqldb.RowScanner for a sql.Row
type rowScanner struct {
	query            string // for error wrapping
	rows             *sql.Rows
	structFieldNamer sqldb.StructFieldNamer
}

func (s *rowScanner) Scan(dest ...interface{}) error {
	err := s.rows.Scan(dest...)
	s.rows.Close()
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
		s.rows.Close()
	}()

	return impl.ScanStruct(s.rows, dest, s.structFieldNamer, nil, nil)
}
