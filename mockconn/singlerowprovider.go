package mockconn

import (
	"context"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// SingleRowProvider implements RowsProvider with a single Row
// that will be re-used for every query.
type SingleRowProvider struct {
	Row *Row
}

func (p *SingleRowProvider) QueryRow(structFieldNamer sqldb.StructFieldNamer, query string, args ...interface{}) sqldb.RowScanner {
	return impl.NewRowScanner(impl.RowAsRows(p.Row), structFieldNamer, query, args)
}

func (p *SingleRowProvider) QueryRows(structFieldNamer sqldb.StructFieldNamer, query string, args ...interface{}) sqldb.RowsScanner {
	return impl.NewRowsScanner(context.Background(), NewRows(p.Row), structFieldNamer, query, args)
}
