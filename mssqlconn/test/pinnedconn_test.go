package mssqlconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

// TestPinnedConnection verifies the sqldb.ConnPinner primitive implemented by
// mssqlconn: a pinned Connection runs every query on one dedicated backend
// session and holds session-scoped state (here a session-owned application lock
// via sp_getapplock) until Close returns the session to the pool.
func TestPinnedConnection(t *testing.T) {
	ctx := t.Context()
	conn := connectMSSQL(t)

	// The cross-session and lock-contention assertions below need the pool to
	// hand out a second backend session while the pinned session holds the
	// first, so they require a pool of at least two connections. Skip rather
	// than block if the pool is capped below that (MaxOpenConnections == 0 means
	// unlimited).
	if maxConns := conn.Stats().MaxOpenConnections; maxConns != 0 && maxConns < 2 {
		t.Skip("pinned-connection test requires a pool of at least 2 connections")
	}

	pinner, ok := conn.(sqldb.ConnPinner)
	require.True(t, ok, "mssqlconn connection must implement sqldb.ConnPinner")

	pinned, err := pinner.Conn(ctx)
	require.NoError(t, err)

	// Every query on the pinned Connection runs on the same backend session.
	spid1, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT @@SPID`,
	)
	require.NoError(t, err)
	spid2, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT @@SPID`,
	)
	require.NoError(t, err)
	assert.Equal(t, spid1, spid2, "pinned queries must share one backend session")

	// While the pinned session is checked out, the pool hands out a different
	// backend session for queries on the parent connection.
	poolSPID, err := sqldb.QueryRowAs[int](ctx, conn, nil, conn,
		/*sql*/ `SELECT @@SPID`,
	)
	require.NoError(t, err)
	assert.NotEqual(t, spid1, poolSPID, "pool query must run on a different session than the pinned one")

	// Session-scoped state lives on the pinned session: a session-owned
	// application lock taken on the pinned session blocks any other session from
	// acquiring it. sp_getapplock returns >= 0 when granted, -1 on timeout.
	const lockResource = "pinnedconn_test_lock"
	getLock := /*sql*/ `
		DECLARE @res int;
		EXEC @res = sp_getapplock @Resource = @p1, @LockMode = N'Exclusive', @LockOwner = N'Session', @LockTimeout = 0;
		SELECT @res;`
	locked, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned, getLock, lockResource)
	require.NoError(t, err)
	require.GreaterOrEqual(t, locked, 0, "pinned session must acquire the application lock")

	otherGotLock, err := sqldb.QueryRowAs[int](ctx, conn, nil, conn, getLock, lockResource)
	require.NoError(t, err)
	assert.Equal(t, -1, otherGotLock, "another session must not take the lock held by the pinned session")

	// Releasing the lock and closing the pinned Connection returns its session to
	// the pool. Close must NOT close the underlying *sql.DB.
	_, err = sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `
		DECLARE @res int;
		EXEC @res = sp_releaseapplock @Resource = @p1, @LockOwner = N'Session';
		SELECT @res;`,
		lockResource,
	)
	require.NoError(t, err)
	require.NoError(t, pinned.Close())

	// The parent connection is still usable after the pinned session is closed.
	one, err := sqldb.QueryRowAs[int](ctx, conn, nil, conn,
		/*sql*/ `SELECT 1`,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, one)
}

// TestPinnedConnectionTransactionAndInfo verifies the non-query surface of a
// pinned mssqlconn Connection: it is not itself a transaction, Begin starts a
// real transaction on the same pinned session, and the embedded Information
// methods route through the pinned session.
func TestPinnedConnectionTransactionAndInfo(t *testing.T) {
	ctx := t.Context()
	conn := connectMSSQL(t)

	pinner, ok := conn.(sqldb.ConnPinner)
	require.True(t, ok, "mssqlconn connection must implement sqldb.ConnPinner")

	pinned, err := pinner.Conn(ctx)
	require.NoError(t, err)
	defer pinned.Close() //nolint:errcheck

	// A pinned connection is not itself a transaction.
	assert.False(t, pinned.Transaction().Active(), "pinned connection must not report an active transaction")
	require.ErrorIs(t, pinned.Commit(), sqldb.ErrNotWithinTransaction)
	require.ErrorIs(t, pinned.Rollback(), sqldb.ErrNotWithinTransaction)

	// Begin rejects a zero transaction ID.
	_, err = pinned.Begin(ctx, 0, nil)
	require.Error(t, err, "Begin must reject transaction ID 0")

	// Begin starts a real transaction on the SAME pinned session.
	pinnedSPID, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT @@SPID`,
	)
	require.NoError(t, err)
	tx, err := pinned.Begin(ctx, 1, nil)
	require.NoError(t, err)
	txSPID, err := sqldb.QueryRowAs[int](ctx, tx, nil, tx,
		/*sql*/ `SELECT @@SPID`,
	)
	require.NoError(t, err)
	assert.Equal(t, pinnedSPID, txSPID, "transaction from a pinned connection must run on the pinned session")
	require.NoError(t, tx.Commit())

	// The Information interface embedded in Connection routes through the
	// pinned session without error.
	_, err = pinned.CurrentSchema(ctx)
	require.NoError(t, err)
}
