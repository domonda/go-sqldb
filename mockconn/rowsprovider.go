package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
)

type RowsProvider interface {
	QueryRow(structFieldNamer sqldb.StructFieldMapper, query string, args ...any) sqldb.RowScanner
	QueryRows(structFieldNamer sqldb.StructFieldMapper, query string, args ...any) sqldb.RowsScanner
}
