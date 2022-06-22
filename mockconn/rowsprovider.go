package mockconn

import (
	sqldb "github.com/domonda/go-sqldb"
)

type RowsProvider interface {
	QueryRow(query string, args ...any) sqldb.Row
	QueryRows(query string, args ...any) sqldb.Rows
}
