package db_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
	"github.com/domonda/go-sqldb/postgres"
)

// testContext returns a context with the given connection
// and the postgres QueryBuilder (which implements QueryBuilder,
// UpsertQueryBuilder, and ReturningQueryBuilder) and StructReflector,
// so that tests don't depend on global state.
func testContext(t *testing.T, conn sqldb.Connection) context.Context {
	t.Helper()
	ctx := db.ContextWithConn(t.Context(), conn)
	ctx = db.ContextWithQueryBuilder(ctx, postgres.QueryBuilder{})
	ctx = db.ContextWithStructReflector(ctx, sqldb.NewTaggedStructReflector())
	return ctx
}

// assertArgs is a test helper for comparing argument slices using fmt.Sprint
// to handle type differences (e.g., int vs int64).
func assertArgs(t *testing.T, got, want []any) {
	t.Helper()
	require.Equal(t, len(want), len(got), "args length")
	for i := range want {
		if fmt.Sprint(got[i]) != fmt.Sprint(want[i]) {
			t.Errorf("args[%d] = %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
		}
	}
}
