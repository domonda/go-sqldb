package mockconn

import (
	"context"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// NewSingleRowProvider a RowsProvider implementation
// with a single row that will be re-used for every query.
func NewSingleRowProvider(row *Row) RowsProvider {
	return &singleRowProvider{row: row, argFmt: DefaultArgFmt}
}

// SingleRowProvider implements RowsProvider with a single Row
// that will be re-used for every query.
type singleRowProvider struct {
	row    *Row
	argFmt string
}

func (p *singleRowProvider) QueryRow(structFieldNamer sqldb.StructFieldNamer, query string, args ...any) sqldb.RowScanner {
	return impl.NewRowScanner(impl.RowAsRows(p.row), structFieldNamer, query, p.argFmt, args)
}

func (p *singleRowProvider) QueryRows(structFieldNamer sqldb.StructFieldNamer, query string, args ...any) sqldb.RowsScanner {
	return impl.NewRowsScanner(context.Background(), NewRows(p.row), structFieldNamer, query, p.argFmt, args)
}
