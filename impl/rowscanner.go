package impl

import (
	"database/sql"

	sqldb "github.com/domonda/go-sqldb"
)

var (
	_ sqldb.RowScanner = &RowScanner{}
	_ sqldb.RowScanner = CurrentRowScanner{}
	_ sqldb.RowScanner = SingleRowScanner{}
)

// RowScanner implements sqldb.RowScanner for a sql.Row
type RowScanner struct {
	rows             Rows
	structFieldNamer sqldb.StructFieldNamer
	query            string // for error wrapping
	argFmt           string // for error wrapping
	args             []any  // for error wrapping
}

func NewRowScanner(rows Rows, structFieldNamer sqldb.StructFieldNamer, query, argFmt string, args []any) *RowScanner {
	return &RowScanner{rows, structFieldNamer, query, argFmt, args}
}

func (s *RowScanner) Scan(dest ...any) (err error) {
	defer func() {
		err = combineErrors(err, s.rows.Close())
		err = WrapNonNilErrorWithQuery(err, s.query, s.argFmt, s.args)
	}()

	if s.rows.Err() != nil {
		return s.rows.Err()
	}
	if !s.rows.Next() {
		if s.rows.Err() != nil {
			return s.rows.Err()
		}
		return sql.ErrNoRows
	}

	return s.rows.Scan(dest...)
}

func (s *RowScanner) ScanStruct(dest any) (err error) {
	defer func() {
		err = combineErrors(err, s.rows.Close())
		err = WrapNonNilErrorWithQuery(err, s.query, s.argFmt, s.args)
	}()

	if s.rows.Err() != nil {
		return s.rows.Err()
	}
	if !s.rows.Next() {
		if s.rows.Err() != nil {
			return s.rows.Err()
		}
		return sql.ErrNoRows
	}

	return ScanStruct(s.rows, dest, s.structFieldNamer, nil, nil)
}

func (s *RowScanner) ScanValues() ([]any, error) {
	return ScanValues(s.rows)
}

func (s *RowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.rows)
}

// CurrentRowScanner calls Rows.Scan without Rows.Next and Rows.Close
type CurrentRowScanner struct {
	Rows             Rows
	StructFieldNamer sqldb.StructFieldNamer
}

func (s CurrentRowScanner) Scan(dest ...any) error {
	return s.Rows.Scan(dest...)
}

func (s CurrentRowScanner) ScanStruct(dest any) error {
	return ScanStruct(s.Rows, dest, s.StructFieldNamer, nil, nil)
}

func (s CurrentRowScanner) ScanValues() ([]any, error) {
	return ScanValues(s.Rows)
}

func (s CurrentRowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.Rows)
}

// SingleRowScanner always uses the same Row
type SingleRowScanner struct {
	Row              Row
	StructFieldNamer sqldb.StructFieldNamer
}

func (s SingleRowScanner) Scan(dest ...any) error {
	return s.Row.Scan(dest...)
}

func (s SingleRowScanner) ScanStruct(dest any) error {
	return ScanStruct(s.Row, dest, s.StructFieldNamer, nil, nil)
}

func (s SingleRowScanner) ScanValues() ([]any, error) {
	return ScanValues(s.Row)
}

func (s SingleRowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.Row)
}
