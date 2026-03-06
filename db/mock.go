package db

import (
	"context"

	sqldb "github.com/domonda/go-sqldb"
)

// NewMockConn returns a new MockConn with the QueryFormatter
// from the ConnExt in the context or the global ConnExt.
func NewMockConn(ctx context.Context) *sqldb.MockConn {
	var queryFormatter sqldb.QueryFormatter = Conn(ctx)
	return sqldb.NewMockConn(queryFormatter)
}

// NewMockStructRows returns a new MockStructRows with column names
// derived from the struct fields of S using the StructReflector
// from the ConnExt in the context or the global ConnExt and the given rows as data.
// Panics if S is not a struct or has no mapped columns.
func NewMockStructRows[S any](ctx context.Context, rows ...S) *sqldb.MockStructRows[S] {
	var reflector sqldb.StructReflector = Conn(ctx)
	return sqldb.NewMockStructRows(reflector, rows...)
}
