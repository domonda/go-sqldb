package sqldb

import (
	"database/sql"

	"github.com/domonda/go-sqldb/reflection"
)

// RowScanner scans the values from a single row.
type RowScanner interface {
	// Scan values of a row into dest variables, which must be passed as pointers.
	Scan(dest ...any) error

	// ScanStruct scans values of a row into a dest struct which must be passed as pointer.
	ScanStruct(dest any) error

	// ScanValues returns the values of a row exactly how they are
	// passed from the database driver to an sql.Scanner.
	// Byte slices will be copied.
	ScanValues() ([]any, error)

	// ScanStrings scans the values of a row as strings.
	// Byte slices will be interpreted as strings,
	// nil (SQL NULL) will be converted to an empty string,
	// all other types are converted with fmt.Sprint(src).
	ScanStrings() ([]string, error)

	// Columns returns the column names.
	Columns() ([]string, error)
}

var (
	_ RowScanner = &rowScanner{}
	_ RowScanner = CurrentRowScanner{}
	_ RowScanner = SingleRowScanner{}
)

// rowScanner implements rowScanner for a sql.Row
type rowScanner struct {
	rows              Rows
	structFieldMapper reflection.StructFieldMapper
	query             string                    // for error wrapping
	argFmt            ParamPlaceholderFormatter // for error wrapping
	args              []any                     // for error wrapping
}

func NewRowScanner(rows Rows, structFieldMapper reflection.StructFieldMapper, query string, argFmt ParamPlaceholderFormatter, args []any) *rowScanner {
	return &rowScanner{rows, structFieldMapper, query, argFmt, args}
}

func (s *rowScanner) Scan(dest ...any) (err error) {
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

func (s *rowScanner) ScanStruct(dest any) (err error) {
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

	return reflection.ScanStruct(s.rows, dest, s.structFieldMapper)
}

func (s *rowScanner) ScanValues() ([]any, error) {
	return ScanValues(s.rows)
}

func (s *rowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.rows)
}

func (s *rowScanner) Columns() ([]string, error) {
	return s.rows.Columns()
}

// CurrentRowScanner calls Rows.Scan without Rows.Next and Rows.Close
type CurrentRowScanner struct {
	Rows              Rows
	StructFieldMapper reflection.StructFieldMapper
}

func (s CurrentRowScanner) Scan(dest ...any) error {
	return s.Rows.Scan(dest...)
}

func (s CurrentRowScanner) ScanStruct(dest any) error {
	return reflection.ScanStruct(s.Rows, dest, s.StructFieldMapper)
}

func (s CurrentRowScanner) ScanValues() ([]any, error) {
	return ScanValues(s.Rows)
}

func (s CurrentRowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.Rows)
}

func (s CurrentRowScanner) Columns() ([]string, error) {
	return s.Rows.Columns()
}

// SingleRowScanner always uses the same Row
type SingleRowScanner struct {
	Row               Row
	StructFieldMapper reflection.StructFieldMapper
}

func (s SingleRowScanner) Scan(dest ...any) error {
	return s.Row.Scan(dest...)
}

func (s SingleRowScanner) ScanStruct(dest any) error {
	return reflection.ScanStruct(s.Row, dest, s.StructFieldMapper)
}

func (s SingleRowScanner) ScanValues() ([]any, error) {
	return ScanValues(s.Row)
}

func (s SingleRowScanner) ScanStrings() ([]string, error) {
	return ScanStrings(s.Row)
}

func (s SingleRowScanner) Columns() ([]string, error) {
	return s.Row.Columns()
}
