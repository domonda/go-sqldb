package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
)

type RowsProvider interface {
	Query(structFieldNamer sqldb.StructReflector, query string, args ...any) (sqldb.Rows, error)
	QueryRow(structFieldNamer sqldb.StructReflector, query string, args ...any) sqldb.RowScanner
	QueryRows(structFieldNamer sqldb.StructReflector, query string, args ...any) sqldb.RowsScanner
}
