package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

// TestPinnedConnUnsupported verifies that db.PinnedConn returns an error
// wrapping errors.ErrUnsupported (and does not run the callback) when the
// connection's driver does not implement sqldb.ConnPinner.
func TestPinnedConnUnsupported(t *testing.T) {
	ctx := db.ContextWithConn(t.Context(), sqldb.NewMockConn(sqldb.NewQueryFormatter("$")))

	called := false
	err := db.PinnedConn(ctx, func(context.Context) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, errors.ErrUnsupported)
	assert.False(t, called, "callback must not run when pinning is unsupported")
}

// fakePinnedConn is a sqldb.Connection that reports itself as already pinned to
// one dedicated session (implements sqldb.PinnedConnection) without being a
// sqldb.ConnPinner — exactly like a real driver's pinned connection.
type fakePinnedConn struct {
	*sqldb.MockConn
}

func (fakePinnedConn) IsPinnedConnection() bool { return true }

// TestPinnedConnAlreadyPinnedPassThrough verifies that db.PinnedConn passes an
// already-pinned connection through unchanged instead of failing with
// ErrUnsupported. A pinned connection is intentionally not a ConnPinner, so
// without the sqldb.PinnedConnection check it would be treated as an
// unsupported driver and the callback would never run.
func TestPinnedConnAlreadyPinnedPassThrough(t *testing.T) {
	pinned := fakePinnedConn{sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))}
	require.False(t, pinned.Transaction().Active())
	_, isPinner := sqldb.Connection(pinned).(sqldb.ConnPinner)
	require.False(t, isPinner, "a pinned connection must not be a ConnPinner")

	ctx := db.ContextWithConn(t.Context(), pinned)
	var got sqldb.Connection
	err := db.PinnedConn(ctx, func(ctx context.Context) error {
		got = db.Conn(ctx)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, sqldb.Connection(pinned), got, "PinnedConn must pass an already-pinned connection through unchanged")
}

// TestPinnedConnTransactionPassThrough verifies that db.PinnedConn passes a
// connection that is already within a transaction through unchanged, because a
// transaction is already bound to a single session.
func TestPinnedConnTransactionPassThrough(t *testing.T) {
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	tx, err := mock.Begin(t.Context(), 1, nil)
	require.NoError(t, err)
	require.True(t, tx.Transaction().Active())

	ctx := db.ContextWithConn(t.Context(), tx)
	var got sqldb.Connection
	err = db.PinnedConn(ctx, func(ctx context.Context) error {
		got = db.Conn(ctx)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, tx, got, "PinnedConn must pass the existing transaction through unchanged")
}
