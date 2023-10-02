package impl

import (
	sqldb "github.com/domonda/go-sqldb"
)

var (
	_ sqldb.RowScanner = CurrentRowScanner{}
	// _ sqldb.RowScanner = SingleRowScanner{}
)

// CurrentRowScanner calls Rows.Scan without Rows.Next and Rows.Close
type CurrentRowScanner struct {
	Rows              Rows
	StructFieldMapper sqldb.StructFieldMapper
}

func (s CurrentRowScanner) Scan(dest ...any) error {
	return ScanRowWithArrays(s.Rows, dest)
}

func (s CurrentRowScanner) ScanStruct(dest any) error {
	return ScanStruct(s.Rows, dest, s.StructFieldMapper)
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
// type SingleRowScanner struct {
// 	Row               Row
// 	StructFieldMapper sqldb.StructFieldMapper
// }

// func (s SingleRowScanner) Scan(dest ...any) error {
// 	return s.Row.Scan(dest...)
// }

// func (s SingleRowScanner) ScanStruct(dest any) error {
// 	return ScanStruct(s.Row, dest, s.StructFieldMapper)
// }

// func (s SingleRowScanner) ScanValues() ([]any, error) {
// 	return ScanValues(s.Row)
// }

// func (s SingleRowScanner) ScanStrings() ([]string, error) {
// 	return ScanStrings(s.Row)
// }

// func (s SingleRowScanner) Columns() ([]string, error) {
// 	return s.Row.Columns()
// }
