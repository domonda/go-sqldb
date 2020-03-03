package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// SingleRowProvider implements RowsProvider with a single Row
// that will be re-used for every query.
type SingleRowProvider struct {
	Row *Row
}

func (p *SingleRowProvider) QueryRow(structFieldNamer sqldb.StructFieldNamer, query string, args ...interface{}) sqldb.RowScanner {
	return &impl.RowScanner{Query: query, Rows: impl.RowAsRows(p.Row), StructFieldNamer: structFieldNamer}
}

func (p *SingleRowProvider) QueryRows(structFieldNamer sqldb.StructFieldNamer, query string, args ...interface{}) sqldb.RowsScanner {
	return &impl.RowsScanner{Query: query, Rows: NewRows(p.Row), StructFieldNamer: structFieldNamer}
}
