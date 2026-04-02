package sqldb

import (
	"context"
	"database/sql"
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
	Close() error
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
