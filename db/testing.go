package db

import (
	"context"
	"testing"

	"github.com/domonda/go-sqldb"
)

// ContextWithNonConnectionForTest returns a new context with a sqldb.Connection
// intended for unit tests that should work without an actual database connection
// by mocking any SQL related functionality so that the connection won't be used.
//
// The transaction related methods of that connection
// simulate a transaction without any actual transaction handling.
// All other methods except Close will cause the test to fail.
func ContextWithNonConnectionForTest(ctx context.Context, t *testing.T) context.Context {
	return ContextWithConn(ctx, sqldb.NonConnForTest(t))
}
