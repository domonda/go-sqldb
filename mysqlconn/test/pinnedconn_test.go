package mysqlconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

// TestPinnedConnection verifies the sqldb.ConnPinner primitive implemented by
// mysqlconn: a pinned Connection runs every query on one dedicated backend
// session and holds session-scoped state (here a session-level named lock via
// GET_LOCK) until Close returns the session to the pool.
func TestPinnedConnection(t *testing.T) {
	ctx := t.Context()
	conn := connectMySQL(t)

	// The cross-session and lock-contention assertions below need the pool to
	// hand out a second backend session while the pinned session holds the
	// first, so they require a pool of at least two connections. Skip rather
	// than block if the pool is capped below that (MaxOpenConnections == 0 means
	// unlimited).
	if maxConns := conn.Stats().MaxOpenConnections; maxConns != 0 && maxConns < 2 {
		t.Skip("pinned-connection test requires a pool of at least 2 connections")
	}

	pinner, ok := conn.(sqldb.ConnPinner)
	require.True(t, ok, "mysqlconn connection must implement sqldb.ConnPinner")

	pinned, err := pinner.Conn(ctx)
	require.NoError(t, err)

	// Every query on the pinned Connection runs on the same backend session.
	id1, err := sqldb.QueryRowAs[uint64](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT CONNECTION_ID()`,
	)
	require.NoError(t, err)
	id2, err := sqldb.QueryRowAs[uint64](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT CONNECTION_ID()`,
	)
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "pinned queries must share one backend session")

	// While the pinned session is checked out, the pool hands out a different
	// backend session for queries on the parent connection.
	poolID, err := sqldb.QueryRowAs[uint64](ctx, conn, nil, conn,
		/*sql*/ `SELECT CONNECTION_ID()`,
	)
	require.NoError(t, err)
	assert.NotEqual(t, id1, poolID, "pool query must run on a different session than the pinned one")

	// Session-scoped state lives on the pinned session: a session-level named
	// lock taken on the pinned session blocks any other session from acquiring it.
	const lockName = "pinnedconn_test_lock"
	locked, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT GET_LOCK(?, 0)`,
		lockName,
	)
	require.NoError(t, err)
	require.Equal(t, 1, locked, "pinned session must acquire the named lock")

	otherGotLock, err := sqldb.QueryRowAs[int](ctx, conn, nil, conn,
		/*sql*/ `SELECT GET_LOCK(?, 0)`,
		lockName,
	)
	require.NoError(t, err)
	assert.Equal(t, 0, otherGotLock, "another session must not take the lock held by the pinned session")

	// Release the named lock explicitly before returning the session to the
	// pool. Close returns the pinned session to the pool rather than closing the
	// physical connection, so an un-released session-level lock could survive on
	// the pooled session and poison later tests.
	released, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT RELEASE_LOCK(?)`,
		lockName,
	)
	require.NoError(t, err)
	require.Equal(t, 1, released, "pinned session must release the named lock it holds")

	// Closing the pinned Connection returns its session to the pool. It must NOT
	// close the underlying *sql.DB, so the parent connection stays usable.
	require.NoError(t, pinned.Close())

	// The parent connection is still usable after the pinned session is closed.
	one, err := sqldb.QueryRowAs[int](ctx, conn, nil, conn,
		/*sql*/ `SELECT 1`,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, one)
}

// TestPinnedConnectionTransactionAndInfo verifies the non-query surface of a
// pinned mysqlconn Connection: it is not itself a transaction, Begin starts a
// real transaction on the same pinned session, and the embedded Information
// methods route through the pinned session.
func TestPinnedConnectionTransactionAndInfo(t *testing.T) {
	ctx := t.Context()
	conn := connectMySQL(t)

	pinner, ok := conn.(sqldb.ConnPinner)
	require.True(t, ok, "mysqlconn connection must implement sqldb.ConnPinner")

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
	pinnedID, err := sqldb.QueryRowAs[uint64](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT CONNECTION_ID()`,
	)
	require.NoError(t, err)
	tx, err := pinned.Begin(ctx, 1, nil)
	require.NoError(t, err)
	txID, err := sqldb.QueryRowAs[uint64](ctx, tx, nil, tx,
		/*sql*/ `SELECT CONNECTION_ID()`,
	)
	require.NoError(t, err)
	assert.Equal(t, pinnedID, txID, "transaction from a pinned connection must run on the pinned session")
	require.NoError(t, tx.Commit())

	// The Information interface embedded in Connection routes through the
	// pinned session without error.
	_, err = pinned.CurrentSchema(ctx)
	require.NoError(t, err)
}
