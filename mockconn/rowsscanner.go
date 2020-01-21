package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
)

// rowsScanner implements sqldb.RowsScanner for sqlx.Rows
type rowsScanner struct{}

func (s *rowsScanner) ScanSlice(dest interface{}) error {
	return ErrMockedScan
}

func (s *rowsScanner) ScanStructSlice(dest interface{}) error {
	return ErrMockedScan
}

func (s *rowsScanner) ForEachRow(callback func(sqldb.RowScanner) error) error {
	return ErrMockedScan
}
