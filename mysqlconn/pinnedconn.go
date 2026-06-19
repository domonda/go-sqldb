package mysqlconn

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/domonda/go-sqldb"
)

// Compile-time check that the mysqlconn connection implements [sqldb.ConnPinner].
// That ConnPinner.Conn returns [sqldb.PinnedConnection] makes the pinnedConn
// implementation of that interface (and therefore [sqldb.Connection]) a
// compile-time guarantee of the method signature below.
var _ sqldb.ConnPinner = (*connection)(nil)

// Conn checks out one dedicated session from the underlying *sql.DB pool and
// returns it wrapped as a [sqldb.PinnedConnection] pinned to that session for
// the lifetime of the returned value. It implements [sqldb.ConnPinner].
//
// Every query runs on the same *sql.Conn, and database/sql does not reap
// checked-out sessions (ConnMaxLifetime/ConnMaxIdleTime don't apply), so the
// session survives until Close returns it to the pool. Use this for
// session-scoped state like GET_LOCK that must live and die on one session.
func (conn *connection) Conn(ctx context.Context) (sqldb.PinnedConnection, error) {
	c, err := conn.db.Conn(ctx)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}
	return &pinnedConn{conn.QueryFormatter, conn.QueryBuilder, conn, c}, nil
}

// pinnedConn is a structural clone of [transaction] that delegates queries to a
// dedicated *sql.Conn instead of a *sql.Tx. Close returns the session to the
// pool instead of rolling back. It is not itself a transaction; Begin starts a
// real transaction on the same pinned session, while Commit and Rollback return
// [sqldb.ErrNotWithinTransaction].
type pinnedConn struct {
	QueryFormatter
	QueryBuilder

	// The parent non-pinned connection is needed
	// for Config(), Ping(), and Stats().
	parent *connection
	conn   *sql.Conn
}

func (conn *pinnedConn) Config() *sqldb.Config {
	return conn.parent.config
}

func (conn *pinnedConn) Ping(ctx context.Context, timeout time.Duration) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.conn.PingContext(ctx)
}

func (conn *pinnedConn) Stats() sql.DBStats { return conn.parent.Stats() }

func (conn *pinnedConn) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (conn *pinnedConn) ExecRowsAffected(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := conn.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, wrapKnownErrors(err)
	}
	return result.RowsAffected()
}

func (conn *pinnedConn) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	rows, err := conn.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	return rows
}

func (conn *pinnedConn) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	stmt, err := conn.conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}
	return sqldb.NewStmt(stmt, query, wrapKnownErrors), nil
}

func (*pinnedConn) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelRepeatableRead // MySQL default
}

// Transaction returns the zero TransactionState because a pinned connection
// is not itself a transaction.
func (conn *pinnedConn) Transaction() sqldb.TransactionState {
	return sqldb.TransactionState{
		ID:   0,
		Opts: nil,
	}
}

// Begin starts a real transaction on the same pinned session so that the
// transaction shares the pinned connection's session-scoped state. The caller
// must Commit or Rollback the returned transaction before closing the pinned
// connection; closing with a transaction still open leaks the session (see
// [sqldb.ConnPinner]).
func (conn *pinnedConn) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	tx, err := conn.conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}
	return newTransaction(conn.parent, tx, opts, id), nil
}

func (conn *pinnedConn) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *pinnedConn) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

// Close returns the pinned session to the pool. It does NOT close the
// underlying *sql.DB.
func (conn *pinnedConn) Close() error {
	return conn.conn.Close()
}

// IsPinnedConnection marks this Connection as already pinned to a single
// dedicated session; see [sqldb.PinnedConnection].
func (conn *pinnedConn) IsPinnedConnection() bool { return true }
