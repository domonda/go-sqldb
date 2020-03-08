package impl

import (
	"database/sql"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

// RowScanner implements sqldb.RowScanner for a sql.Row
type RowScanner struct {
	Query            string // for error wrapping
	Rows             Rows
	StructFieldNamer sqldb.StructFieldNamer
}

func (s *RowScanner) Scan(dest ...interface{}) (err error) {
	if s.Rows.Err() != nil {
		return s.Rows.Err()
	}
	if !s.Rows.Next() {
		if s.Rows.Err() != nil {
			return s.Rows.Err()
		}
		return sql.ErrNoRows
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("query `%s` returned error: %w", s.Query, err)
		}
		s.Rows.Close()
	}()

	return s.Rows.Scan(dest...)
}

func (s *RowScanner) ScanStruct(dest interface{}) (err error) {
	if s.Rows.Err() != nil {
		return s.Rows.Err()
	}
	if !s.Rows.Next() {
		if s.Rows.Err() != nil {
			return s.Rows.Err()
		}
		return sql.ErrNoRows
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("query `%s` returned error: %w", s.Query, err)
		}
		s.Rows.Close()
	}()

	return ScanStruct(s.Rows, dest, s.StructFieldNamer, nil, nil)
}

func (s *RowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.Rows)
}

// CurrentRowScanner calls Rows.Scan without Rows.Next and Rows.Close
type CurrentRowScanner struct {
	Rows             Rows
	StructFieldNamer sqldb.StructFieldNamer
}

func (s CurrentRowScanner) Scan(dest ...interface{}) error {
	return s.Rows.Scan(dest...)
}

func (s CurrentRowScanner) ScanStruct(dest interface{}) error {
	return ScanStruct(s.Rows, dest, s.StructFieldNamer, nil, nil)
}

func (s CurrentRowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.Rows)
}

// SingleRowScanner always uses the same Row
type SingleRowScanner struct {
	Row              Row
	StructFieldNamer sqldb.StructFieldNamer
}

func (s SingleRowScanner) Scan(dest ...interface{}) error {
	return s.Row.Scan(dest...)
}

func (s SingleRowScanner) ScanStruct(dest interface{}) error {
	return ScanStruct(s.Row, dest, s.StructFieldNamer, nil, nil)
}

func (s SingleRowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.Row)
}
