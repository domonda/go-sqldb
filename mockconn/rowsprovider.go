package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
)

type RowsProvider interface {
	QueryRow(structFieldNamer sqldb.StructFieldNamer, query string, args ...interface{}) sqldb.RowScanner
	QueryRows(structFieldNamer sqldb.StructFieldNamer, query string, args ...interface{}) sqldb.RowsScanner
}
