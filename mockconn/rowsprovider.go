package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
)

type RowsProvider interface {
	QueryRow(structFieldNamer sqldb.StructFieldNamer, query string, args ...any) sqldb.RowScanner
	QueryRows(structFieldNamer sqldb.StructFieldNamer, query string, args ...any) sqldb.RowsScanner
}
