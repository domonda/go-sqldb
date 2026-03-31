package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

func TestContextWithConn(t *testing.T) {
	conn := new(sqldb.MockConn)
	ctx := db.ContextWithConn(t.Context(), conn)

	got := db.Conn(ctx)
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
		db.SetConn(conn)
		t.Cleanup(func() { db.SetConn(sqldb.NewErrConn(sqldb.ErrNoDatabaseConnection)) })

		err := db.Close()
		require.NoError(t, err)
		require.Equal(t, 1, closeCount, "MockClose call count")
	})
}
