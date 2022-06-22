package mockconn

import (
	"context"

	sqldb "github.com/domonda/go-sqldb"
)

// NewSingleRowProvider a RowsProvider implementation
// with a single row that will be re-used for every query.
func NewSingleRowProvider(row *Row) RowsProvider {
	return &singleRowProvider{row: row, argFmt: DefaultParamPlaceholderFormatter}
}

// SingleRowProvider implements RowsProvider with a single Row
// that will be re-used for every query.
type singleRowProvider struct {
	row    *Row
	argFmt sqldb.ParamPlaceholderFormatter
}

func (p *singleRowProvider) QueryRow(query string, args ...any) sqldb.Row {
	return sqldb.NewRow(context.Background(), sqldb.RowAsRows(p.row), query, p.argFmt, args)
}

func (p *singleRowProvider) QueryRows(query string, args ...any) sqldb.Rows {
	return sqldb.NewRows(context.Background(), sqldb.NewRows(p.row), query, p.argFmt, args)
}
