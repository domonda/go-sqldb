package db

import (
	"context"
	"errors"

	"github.com/domonda/go-sqldb"
)

// PinnedConn runs pinnedFunc with a context whose [Conn] is pinned to one
// dedicated database session for the duration of pinnedFunc, then returns that
// session to the pool. Every db function called with the context passed to
// pinnedFunc runs on the same underlying session, which is required for
// session-scoped state like PostgreSQL pg_advisory_lock that must live and die
// on one session:
//
//	err := db.PinnedConn(ctx, func(ctx context.Context) error {
//		if err := db.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
//			return err
//		}
//		defer db.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockID)
//		// ... work that relies on the advisory lock, all on the pinned session ...
//		return nil
//	})
//
// If the connection from ctx is already within a transaction, or is itself
// already a pinned connection, it is already bound to a single session, so
// pinnedFunc is called with ctx unchanged (mirroring how [Transaction] passes
// an existing transaction through). Otherwise the pinned session is returned to
// the pool when pinnedFunc returns, even if it panics.
//
// Inside pinnedFunc the context connection is the pinned session, so
// [Transaction], [TransactionSavepoint], and [IsolatedTransaction] all begin on
// it and run on the pinned session, seeing its session-scoped state (advisory
// locks, SET SESSION settings, temporary tables). As without pinning, an
// [IsolatedTransaction] started from within another transaction still begins on
// a separate pool session.
//
// Outside the passthrough cases above (no active transaction and not already
// pinned), PinnedConn checks out a pinned session and returns an error wrapping
// errors.ErrUnsupported if the connection's driver does not implement
// [sqldb.ConnPinner]. The pqconn, mysqlconn, mssqlconn, and oraconn drivers
// implement it; sqliteconn does not, having no connection pool.
//
// When the pinned session has to outlive a single callback, use
// [sqldb.PinConn] directly to obtain a pinned [sqldb.Connection] that the
// caller must Close explicitly:
//
//	pinned, err := sqldb.PinConn(ctx, db.Conn(ctx))
//	if err != nil {
//		return err
//	}
//	defer pinned.Close() // returns the session to the pool
//	ctx = db.ContextWithConn(ctx, pinned) // route db functions to the pinned session
func PinnedConn(ctx context.Context, pinnedFunc func(context.Context) error) (err error) {
	conn := Conn(ctx)
	// A transaction or an already-pinned connection is already bound to one
	// dedicated session; pass it through unchanged instead of checking out a
	// second, unrelated pool session.
	if conn.Transaction().Active() {
		return pinnedFunc(ctx)
	}
	if _, alreadyPinned := conn.(sqldb.PinnedConnection); alreadyPinned {
		return pinnedFunc(ctx)
	}
	pinned, err := sqldb.PinConn(ctx, conn)
	if err != nil {
		return err
	}
	defer func() {
		// Return the session to the pool, joining any Close error with the
		// error from pinnedFunc; errors.Join drops nil operands, so a Close
		// error surfaces whether or not pinnedFunc itself failed.
		if closeErr := pinned.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	return pinnedFunc(ContextWithConn(ctx, pinned))
}

// PinnedConnResult is like [PinnedConn] but returns the result of pinnedFunc.
func PinnedConnResult[T any](ctx context.Context, pinnedFunc func(context.Context) (T, error)) (result T, err error) {
	err = PinnedConn(ctx, func(ctx context.Context) error {
		result, err = pinnedFunc(ctx)
		return err
	})
	return result, err
}
