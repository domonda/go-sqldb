package pqconn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

// TestPinnedConnection verifies the sqldb.ConnPinner primitive implemented by
// pqconn: a pinned Connection runs every query on one dedicated backend session
// and holds session-scoped state (here a session-level advisory lock) until
// Close returns the session to the pool.
func TestPinnedConnection(t *testing.T) {
	ctx := t.Context()
	conn := pqConnect(t)

	// The cross-session and lock-contention assertions below need the pool to
	// hand out a second backend session while the pinned session holds the
	// first, so they require a pool of at least two connections. Skip rather
	// than block if the pool is capped below that (MaxOpenConnections == 0 means
	// unlimited).
	if maxConns := conn.Stats().MaxOpenConnections; maxConns != 0 && maxConns < 2 {
		t.Skip("pinned-connection test requires a pool of at least 2 connections")
	}

	pinner, ok := conn.(sqldb.ConnPinner)
	require.True(t, ok, "pqconn connection must implement sqldb.ConnPinner")

	pinned, err := pinner.Conn(ctx)
	require.NoError(t, err)

	// Every query on the pinned Connection runs on the same backend session.
	pid1, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT pg_backend_pid()`,
	)
	require.NoError(t, err)
	pid2, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT pg_backend_pid()`,
	)
	require.NoError(t, err)
	assert.Equal(t, pid1, pid2, "pinned queries must share one backend session")

	// While the pinned session is checked out, the pool hands out a different
	// backend session for queries on the parent connection.
	poolPID, err := sqldb.QueryRowAs[int](ctx, conn, nil, conn,
		/*sql*/ `SELECT pg_backend_pid()`,
	)
	require.NoError(t, err)
	assert.NotEqual(t, pid1, poolPID, "pool query must run on a different session than the pinned one")

	// Session-scoped state lives on the pinned session: a session-level advisory
	// lock taken on the pinned session blocks any other session from acquiring it.
	const lockKey = 918273645
	locked, err := sqldb.QueryRowAs[bool](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT pg_try_advisory_lock($1)`,
		lockKey,
	)
	require.NoError(t, err)
	require.True(t, locked, "pinned session must acquire the advisory lock")

	otherGotLock, err := sqldb.QueryRowAs[bool](ctx, conn, nil, conn,
		/*sql*/ `SELECT pg_try_advisory_lock($1)`,
		lockKey,
	)
	require.NoError(t, err)
	assert.False(t, otherGotLock, "another session must not take the lock held by the pinned session")

	// Closing the pinned Connection returns its session to the pool and releases
	// the session-scoped advisory lock. It must NOT close the underlying *sql.DB.
	require.NoError(t, pinned.Close())

	gotLockAfterClose, err := sqldb.QueryRowAs[bool](ctx, conn, nil, conn,
		/*sql*/ `SELECT pg_try_advisory_lock($1)`,
		lockKey,
	)
	require.NoError(t, err)
	assert.True(t, gotLockAfterClose, "advisory lock must be released once the pinned session is closed")

	// The parent connection is still usable after the pinned session is closed.
	_, err = sqldb.QueryRowAs[bool](ctx, conn, nil, conn,
		/*sql*/ `SELECT pg_advisory_unlock($1)`,
		lockKey,
	)
	require.NoError(t, err)
}

// TestPinConnHelper verifies the sqldb.PinConn helper and that a transaction
// is intentionally not a ConnPinner.
func TestPinConnHelper(t *testing.T) {
	ctx := t.Context()
	conn := pqConnect(t)

	// Happy path: PinConn checks out a dedicated session on the pool connection.
	pinned, err := sqldb.PinConn(ctx, conn)
	require.NoError(t, err)
	pid1, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT pg_backend_pid()`,
	)
	require.NoError(t, err)
	pid2, err := sqldb.QueryRowAs[int](ctx, pinned, nil, pinned,
		/*sql*/ `SELECT pg_backend_pid()`,
	)
	require.NoError(t, err)
	assert.Equal(t, pid1, pid2, "PinConn must return a session-pinned connection")
	require.NoError(t, pinned.Close())

	// Begin on the pinned connection runs the transaction on the same session.
	pinned2, err := sqldb.PinConn(ctx, conn)
	require.NoError(t, err)
	defer pinned2.Close()
	pinnedPID, err := sqldb.QueryRowAs[int](ctx, pinned2, nil, pinned2,
		/*sql*/ `SELECT pg_backend_pid()`,
	)
	require.NoError(t, err)
	tx, err := pinned2.Begin(ctx, 1, nil)
	require.NoError(t, err)
	txPID, err := sqldb.QueryRowAs[int](ctx, tx, nil, tx,
		/*sql*/ `SELECT pg_backend_pid()`,
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	assert.Equal(t, pinnedPID, txPID, "a transaction begun on a pinned connection runs on the pinned session")

	// A transaction is intentionally not a ConnPinner; PinConn refuses it.
	poolTx, err := conn.Begin(ctx, 2, nil)
	require.NoError(t, err)
	defer poolTx.Rollback()
	_, ok := poolTx.(sqldb.ConnPinner)
	assert.False(t, ok, "a transaction must not implement sqldb.ConnPinner")
	_, err = sqldb.PinConn(ctx, poolTx)
	require.ErrorIs(t, err, sqldb.ErrWithinTransaction)
}

// TestDBPinnedConn verifies the db.PinnedConn helper: db.* calls inside the
// callback share one pinned session, and within a transaction the callback runs
// on the transaction's own session.
func TestDBPinnedConn(t *testing.T) {
	ctx := db.ContextWithConn(t.Context(), pqConnect(t))

	// Happy path: db.* calls inside the callback all run on one pinned session.
	pids, err := db.PinnedConnResult(ctx, func(ctx context.Context) ([2]int, error) {
		p1, err := db.QueryRowAs[int](ctx,
			/*sql*/ `SELECT pg_backend_pid()`,
		)
		if err != nil {
			return [2]int{}, err
		}
		p2, err := db.QueryRowAs[int](ctx,
			/*sql*/ `SELECT pg_backend_pid()`,
		)
		return [2]int{p1, p2}, err
	})
	require.NoError(t, err)
	assert.Equal(t, pids[0], pids[1], "db.* calls inside PinnedConn must share one session")

	// Nested PinnedConn passes the already-pinned connection through: the inner
	// callback runs on the same pinned session instead of failing because a
	// pinned connection is intentionally not a ConnPinner.
	err = db.PinnedConn(ctx, func(ctx context.Context) error {
		outerPID, err := db.QueryRowAs[int](ctx,
			/*sql*/ `SELECT pg_backend_pid()`,
		)
		if err != nil {
			return err
		}
		return db.PinnedConn(ctx, func(ctx context.Context) error {
			innerPID, err := db.QueryRowAs[int](ctx,
				/*sql*/ `SELECT pg_backend_pid()`,
			)
			if err != nil {
				return err
			}
			assert.Equal(t, outerPID, innerPID, "nested PinnedConn must run on the same pinned session")
			return nil
		})
	})
	require.NoError(t, err)

	// Pass-through: PinnedConn inside a transaction runs on the transaction's
	// own session rather than checking out an unrelated pool session.
	err = db.Transaction(ctx, func(ctx context.Context) error {
		txPID, err := db.QueryRowAs[int](ctx,
			/*sql*/ `SELECT pg_backend_pid()`,
		)
		if err != nil {
			return err
		}
		return db.PinnedConn(ctx, func(ctx context.Context) error {
			pinnedPID, err := db.QueryRowAs[int](ctx,
				/*sql*/ `SELECT pg_backend_pid()`,
			)
			if err != nil {
				return err
			}
			assert.Equal(t, txPID, pinnedPID, "PinnedConn must pass an existing transaction through unchanged")
			return nil
		})
	})
	require.NoError(t, err)
}
