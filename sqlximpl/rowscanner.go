package sqlximpl

import "github.com/jmoiron/sqlx"

type rowScanner struct {
	row *sqlx.Row
}

func (s rowScanner) Scan(dest ...interface{}) error {
	return s.row.Scan(dest...)
}

func (s rowScanner) ScanStruct(dest interface{}) error {
	return s.row.StructScan(dest)
}
