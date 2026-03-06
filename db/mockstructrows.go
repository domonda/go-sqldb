package db

import (
	"context"

	sqldb "github.com/domonda/go-sqldb"
)

// NewMockStructRows returns a new MockStructRows with column names
// derived from the struct fields of S using the StructReflector
// from the connection in the context or the global connection and the given rows as data.
// Panics if S is not a struct or has no mapped columns.
func NewMockStructRows[S any](ctx context.Context, rows ...S) *sqldb.MockStructRows[S] {
	reflector := Conn(ctx).StructReflector
	return sqldb.NewMockStructRows(reflector, rows...)
}
