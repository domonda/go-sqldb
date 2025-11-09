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

type TransactionState struct {
	ID   uint64
	Opts *sql.TxOptions
}

func (ts TransactionState) Active() bool {
	return ts.ID != 0
}

// type Preparer interface {
// 	// Prepare a statement for execution.
// 	Prepare(ctx context.Context, query string) (Stmt, error)
// }

// type Executor interface {
// 	// Exec executes a query with optional args.
// 	Exec(ctx context.Context, query string, args ...any) error
// }

// type Querier interface {
// 	// Query queries rows with optional args.
// 	// Any error will be returned by the Rows.Err method.
// 	Query(ctx context.Context, query string, args ...any) Rows
// }

// Connection represents a database connection or transaction
type Connection interface {
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
	// to aquire a new ID in a threadsafe way.
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

type ListenerConnection interface {
	Connection

	// ListenOnChannel will call onNotify for every channel notification
	// and onUnlisten if the channel gets unlistened
	// or the listener connection gets closed for some reason.
	// It is valid to pass nil for onNotify or onUnlisten to not get those callbacks.
	// Note that the callbacks are called in sequence from a single go routine,
	// so callbacks should offload long running or potentially blocking code to other go routines.
	// Panics from callbacks will be recovered and logged.
	ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error

	// UnlistenChannel will stop listening on the channel.
	// An error is returned, when the channel was not listened to
	// or the listener connection is closed.
	UnlistenChannel(channel string) error

	// IsListeningOnChannel returns if a channel is listened to.
	IsListeningOnChannel(channel string) bool
}
