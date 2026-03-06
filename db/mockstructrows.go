package db

import (
	"context"

	"github.com/domonda/go-sqldb/mockconn"
)

// NewMockStructRows returns a new MockStructRows with column names
// derived from the struct fields of S using the StructFieldMapper
// from the connection in the context or the global connection and the given rows as data.
// Panics if S is not a struct or has no mapped columns.
func NewMockStructRows[S any](ctx context.Context, rows ...S) *mockconn.MockStructRows[S] {
	namer := Conn(ctx).StructFieldMapper()
	return mockconn.NewMockStructRows(namer, rows...)
}
