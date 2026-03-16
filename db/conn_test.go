package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

// testContext returns a context with the given connection
// and the standard QueryBuilder and StructReflector,
// so that tests don't depend on global state.
func testContext(t *testing.T, conn sqldb.Connection) context.Context {
	t.Helper()
	ctx := ContextWithConn(t.Context(), conn)
	ctx = ContextWithQueryBuilder(ctx, sqldb.StdQueryBuilder{})
	ctx = ContextWithStructReflector(ctx, sqldb.NewTaggedStructReflector())
	return ctx
}

func TestContextWithConn(t *testing.T) {
	conn := new(sqldb.MockConn)
	ctx := ContextWithConn(t.Context(), conn)

	got := Conn(ctx)
	require.Equal(t, conn, got)
}

func TestClose(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn := new(sqldb.MockConn)
		var closeCount int
		conn.MockClose = func() error {
			closeCount++
			return nil
		}

		// Save and restore global conn
		saved := globalConn
		defer func() { globalConn = saved }()

		SetConn(conn)
		err := Close()
		require.NoError(t, err)
		require.Equal(t, 1, closeCount, "MockClose call count")
	})
}
