package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
)

type RowsProvider interface {
	Query(structFieldNamer StructReflector, query string, args ...any) (sqldb.Rows, error)
	QueryRow(structFieldNamer StructReflector, query string, args ...any) sqldb.RowScanner
	QueryRows(structFieldNamer StructReflector, query string, args ...any) sqldb.RowsScanner
}
