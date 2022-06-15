package mockconn

import (
	"context"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
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

func (p *singleRowProvider) QueryRow(mapper reflection.StructFieldMapper, query string, args ...any) sqldb.RowScanner {
	return sqldb.NewRowScanner(sqldb.RowAsRows(p.row), mapper, query, p.argFmt, args)
}

func (p *singleRowProvider) QueryRows(mapper reflection.StructFieldMapper, query string, args ...any) sqldb.RowsScanner {
	return sqldb.NewRowsScanner(context.Background(), NewRows(p.row), mapper, query, p.argFmt, args)
}
