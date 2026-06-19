package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type (
	// OnNotifyFunc is a callback type passed to Connection.ListenOnChannel
	OnNotifyFunc func(channel, payload string)

	// OnUnlistenFunc is a callback type passed to Connection.ListenOnChannel
	OnUnlistenFunc func(channel string)
)

// TransactionState holds the state of a database transaction.
// A zero value TransactionState means no active transaction.
type TransactionState struct {
	ID   uint64
	Opts *sql.TxOptions
}

// Active returns true if there is an active transaction (ID != 0).
func (ts TransactionState) Active() bool {
	return ts.ID != 0
}

// Preparer is an interface for preparing SQL statements.
type Preparer interface {
	// Prepare a statement for execution.
	Prepare(ctx context.Context, query string) (Stmt, error)
}

// Executor is an interface for executing SQL statements.
type Executor interface {
	// Exec executes a query with optional args.
	Exec(ctx context.Context, query string, args ...any) error

	// ExecRowsAffected executes a query with optional args
	// and returns the number of rows affected by an
	// update, insert, or delete. Not every database or database
	// driver may support this.
	ExecRowsAffected(ctx context.Context, query string, args ...any) (int64, error)
}

// Querier is an interface for querying rows from the database.
type Querier interface {
	// Query queries rows with optional args.
	// Any error will be returned by the Rows.Err method.
	Query(ctx context.Context, query string, args ...any) Rows
}

// Connection represents a database connection or transaction
type Connection interface {
	QueryFormatter

	Information

	// Config returns the configuration used to establish this connection.
	Config() *Config

	// Stats returns the sql.DBStats of this connection.
	Stats() sql.DBStats

	// Ping returns an error if the database
	// does not answer on this connection
	// with an optional timeout.
	// The passed timeout has to be greater zero
	// to be considered.
	Ping(ctx context.Context, timeout time.Duration) error

	// Prepare a statement for execution.
	Prepare(ctx context.Context, query string) (Stmt, error)

	// Exec executes a query with optional args.
	Exec(ctx context.Context, query string, args ...any) error

	// ExecRowsAffected executes a query with optional args
	// and returns the number of rows affected by an
	// update, insert, or delete. Not every database or database
	// driver may support this.
	ExecRowsAffected(ctx context.Context, query string, args ...any) (int64, error)

	// Query queries rows with optional args.
	// Any error will be returned by the Rows.Err method.
	Query(ctx context.Context, query string, args ...any) Rows

	// DefaultIsolationLevel returns the isolation level of the database
	// driver that is used when no isolation level
	// is specified when beginning a new transaction.
	DefaultIsolationLevel() sql.IsolationLevel

	// Transaction returns the transaction state of the connection
	Transaction() TransactionState

	// Begin a new transaction.
	// If the connection is already a transaction then a brand
	// new transaction will begin based on the connection
	// that started this transaction.
	// The passed id and opts will be returned from the transaction's
	// Connection.Transaction method as TransactionState.
	// Implementations should use the function NextTransactionID
	// to acquire a new ID in a threadsafe way.
	Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error)

	// Commit the current transaction.
	// Returns ErrNotWithinTransaction if the connection
	// is not within a transaction.
	Commit() error

	// Rollback the current transaction.
	// Returns ErrNotWithinTransaction if the connection
	// is not within a transaction.
	Rollback() error

	// Close the connection.
	// Transactions will be rolled back.
	//
	// A [PinnedConnection] (returned by [ConnPinner.Conn]) overrides this to
	// return its dedicated session to the pool instead of closing the
	// underlying *sql.DB or rolling back; see [PinnedConnection].
	Close() error
}

// ConnectionWithoutPlaceholderSubstitution returns a [Connection] that wraps
// conn and overrides [QueryFormatter.SubstitutePlaceholders] to return the
// query unchanged, keeping placeholders intact in error messages, logs, and
// debugging output instead of substituting argument values. Use this to wrap
// a [Connection] whose arguments may contain secrets that must not leak into
// error logs, and pass the result to sqldb functions or to db.SetConn.
//
// The wrapper persists across [Connection.Begin] so that transactions
// started from the wrapped connection inherit the no-substitution behavior.
func ConnectionWithoutPlaceholderSubstitution(conn Connection) Connection {
	return connectionWithoutPlaceholderSubstitution{Connection: conn}
}

type connectionWithoutPlaceholderSubstitution struct {
	Connection
}

func (c connectionWithoutPlaceholderSubstitution) SubstitutePlaceholders(query string, args []any) (string, error) {
	return query, nil
}

// Begin overrides the embedded Connection's Begin so the returned transaction
// stays wrapped: without this override the wrapper would only protect queries
// executed directly on it, while queries inside transactions would substitute
// placeholders normally and could leak secrets to error messages and logs.
func (c connectionWithoutPlaceholderSubstitution) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error) {
	tx, err := c.Connection.Begin(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	return connectionWithoutPlaceholderSubstitution{Connection: tx}, nil
}

// ConnPinner is implemented by Connections that can check out one dedicated
// session pinned for the lifetime of the returned Connection. Every query runs
// on the same underlying *sql.Conn, and database/sql does NOT reap checked-out
// connections (ConnMaxLifetime/ConnMaxIdleTime don't apply). Close() returns it
// to the pool. Required for session-scoped state like pg_advisory_lock that must
// live and die on one session.
//
// Not every driver implements ConnPinner, so type-assert at the call site
// (the same pattern as [UpsertQueryBuilder] and [ReturningQueryBuilder]):
//
//	if pinner, ok := conn.(sqldb.ConnPinner); ok {
//		pinned, err := pinner.Conn(ctx)
//		// ...
//		defer pinned.Close() // returns the session to the pool
//	}
type ConnPinner interface {
	// Conn checks out a dedicated session from the pool and returns a
	// [PinnedConnection] pinned to it for the lifetime of the returned value.
	// The caller must Close the returned Connection to return the
	// session to the pool.
	//
	// Any transaction started with Begin on the returned Connection must be
	// committed or rolled back before Close. Closing while a transaction is
	// still open returns the session to the pool with the transaction still
	// holding it, leaking the session (and any locks or temporary state on
	// it) until its context is canceled or it is garbage collected.
	Conn(ctx context.Context) (PinnedConnection, error)
}

// PinConn checks out one dedicated session from conn and returns a
// [PinnedConnection] pinned to it for the lifetime of the returned value, by
// type-asserting conn to [ConnPinner]. The caller must Close the returned
// Connection to return the session to the pool.
//
// PinConn returns [ErrWithinTransaction] if conn is already within a
// transaction: a transaction is already bound to a single session, and checking
// out a new pool session inside it would silently run on an unrelated session
// (a different backend that does not see the transaction's uncommitted changes
// or hold its locks). To run statements on the transaction's own session, use
// conn directly. PinConn returns an error wrapping [errors.ErrUnsupported] if
// conn's driver does not implement [ConnPinner] (the pqconn, mysqlconn,
// mssqlconn, and oraconn drivers do; sqliteconn does not, having no pool).
func PinConn(ctx context.Context, conn Connection) (PinnedConnection, error) {
	if conn.Transaction().Active() {
		return nil, fmt.Errorf("PinConn: %w", ErrWithinTransaction)
	}
	pinner, ok := conn.(ConnPinner)
	if !ok {
		return nil, fmt.Errorf("PinConn: connection type %T does not implement sqldb.ConnPinner: %w", conn, errors.ErrUnsupported)
	}
	return pinner.Conn(ctx)
}

// PinnedConnection is implemented by the Connection returned from
// [ConnPinner.Conn]: a connection already pinned to a single dedicated session.
// It is a marker that lets helpers recognize an already-pinned connection and
// pass it through unchanged instead of checking out a second, unrelated pool
// session. A pinned connection deliberately does NOT implement [ConnPinner], so
// it cannot be pinned again; the marker is how callers tell "already pinned"
// apart from "driver does not support pinning".
type PinnedConnection interface {
	Connection

	// IsPinnedConnection is a marker method that always returns true. It
	// identifies a Connection already pinned to a single dedicated session.
	IsPinnedConnection() bool
}

// ListenerConnection extends Connection with channel notification support.
type ListenerConnection interface {
	Connection

	// ListenOnChannel will call onNotify for every channel notification
	// and onUnlisten if the channel gets unlistened
	// or the listener connection gets closed for some reason.
	// It is valid to pass nil for onNotify or onUnlisten to not get those callbacks.
	// Calling ListenOnChannel multiple times for the same channel
	// adds additional callbacks; all registered callbacks will be invoked
	// for each notification.
	// Note that callbacks are called concurrently in separate go routines,
	// so callbacks must be safe for concurrent execution.
	// Panics from callbacks will be recovered and logged.
	ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error

	// UnlistenChannel will stop listening on the channel
	// and remove all registered callbacks for it.
	// Registered unlisten callbacks are called concurrently
	// and UnlistenChannel waits for all of them to complete before returning.
	// An error is returned, when the channel was not listened to
	// or the listener connection is closed.
	UnlistenChannel(channel string) error

	// IsListeningOnChannel returns if a channel is listened to.
	IsListeningOnChannel(channel string) bool
}
