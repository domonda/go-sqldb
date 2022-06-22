package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
)

type RowsProvider interface {
	QueryRow(structFieldMapper reflection.StructFieldMapper, query string, args ...any) sqldb.RowScanner
	QueryRows(structFieldMapper reflection.StructFieldMapper, query string, args ...any) sqldb.RowsScanner
}
