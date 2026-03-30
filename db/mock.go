package db

import (
	"context"

	sqldb "github.com/domonda/go-sqldb"
)

// NewMockConn returns a new MockConn with the QueryFormatter
// from the Connection in the context or the global Connection.
func NewMockConn(ctx context.Context) *sqldb.MockConn {
	return sqldb.NewMockConn(Conn(ctx))
}

// NewMockStructRows returns a new MockStructRows with column names
// derived from the struct fields of S using the StructReflector
// from the context or the global StructReflector and the given rows as data.
// Panics if S is not a struct or has no mapped columns.
func NewMockStructRows[S any](ctx context.Context, rows ...S) *sqldb.MockStructRows[S] {
	return sqldb.NewMockStructRows(StructReflector(ctx), rows...)
}
