package sqldb

import (
	"database/sql"
)

// Row is an interface with the methods of sql.Rows
// that are needed for ScanStruct.
// Allows mocking for tests without an SQL driver.
type Row interface {
	// Columns returns the column names.
	Columns() ([]string, error)

	// Scan copies the columns in the current row into the values pointed
	// at by dest. The number of values in dest must be the same as the
	// number of columns in Rows.
	Scan(dest ...any) error
}

// RowScanner scans the values from a single row.
type RowScanner interface {
	// Columns returns the column names.
	Columns() ([]string, error)

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
}

// rowScanner implements RowScanner for Rows
type rowScanner struct {
	row   Rows
	query string     // for error wrapping
	args  []any      // for error wrapping
	conn  Connection // for error wrapping
}

func NewRowScanner(row Rows, query string, args []any, conn Connection) RowScanner {
	return &rowScanner{row: row, query: query, args: args, conn: conn}
}

func (s *rowScanner) Columns() (columns []string, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return s.row.Columns()
}

func (s *rowScanner) Scan(dest ...any) (err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	if s.row.Err() != nil {
		return s.row.Err()
	}
	if !s.row.Next() {
		if s.row.Err() != nil {
			return s.row.Err()
		}
		return sql.ErrNoRows
	}

	return s.row.Scan(dest...)
}

func (s *rowScanner) ScanStruct(dest any) (err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	if s.row.Err() != nil {
		return s.row.Err()
	}
	if !s.row.Next() {
		if s.row.Err() != nil {
			return s.row.Err()
		}
		return sql.ErrNoRows
	}

	return ScanStruct(s.row, dest, s.conn)
}

func (s *rowScanner) ScanValues() (vals []any, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	if s.row.Err() != nil {
		return nil, s.row.Err()
	}
	if !s.row.Next() {
		if s.row.Err() != nil {
			return nil, s.row.Err()
		}
		return nil, sql.ErrNoRows
	}

	return ScanValues(s.row)
}

func (s *rowScanner) ScanStrings() (strs []string, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	if s.row.Err() != nil {
		return nil, s.row.Err()
	}
	if !s.row.Next() {
		if s.row.Err() != nil {
			return nil, s.row.Err()
		}
		return nil, sql.ErrNoRows
	}

	return ScanStrings(s.row)
}

// currentRowScanner implements RowScanner for Row
type currentRowScanner struct {
	row   Row
	query string     // for error wrapping
	args  []any      // for error wrapping
	conn  Connection // for error wrapping
}

func NewCurrentRowScanner(row Row, query string, args []any, conn Connection) RowScanner {
	return &currentRowScanner{row: row, query: query, args: args, conn: conn}
}

func (s *currentRowScanner) Columns() (columns []string, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return s.row.Columns()
}

func (s *currentRowScanner) Scan(dest ...any) (err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return s.row.Scan(dest...)
}

func (s *currentRowScanner) ScanStruct(dest any) (err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return ScanStruct(s.row, dest, s.conn)
}

func (s *currentRowScanner) ScanValues() (vals []any, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return ScanValues(s.row)
}

func (s *currentRowScanner) ScanStrings() (strs []string, err error) {
	defer WrapResultErrorWithQuery(&err, s.query, s.args, s.conn)

	return ScanStrings(s.row)
}
