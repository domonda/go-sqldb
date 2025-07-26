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

type TransactionInfo struct {
	No   uint64
	Opts *sql.TxOptions

	// The default isolation level of the database connection
	// that is used when no isolation level
	// is specified when beginning a new transaction.
	DefaultIsolationLevel sql.IsolationLevel
}

type Queryer interface {
	QueryFormatter

	// Prepare a statement for execution.
	Prepare(ctx context.Context, query string) (Stmt, error)

	// Exec executes a query with optional args.
	Exec(ctx context.Context, query string, args ...any) error

	// Query queries rows with optional args.
	// Any error will be returned by the Rows.Err method.
	Query(ctx context.Context, query string, args ...any) Rows
}

// Connection represents a database connection or transaction
type Connection interface {

	// Ping returns an error if the database
	// does not answer on this connection
	// with an optional timeout.
	// The passed timeout has to be greater zero
	// to be considered.
	Ping(ctx context.Context, timeout time.Duration) error

	// Stats returns the sql.DBStats of this connection.
	Stats() sql.DBStats

	// Config returns the configuration used
	// to create this connection.
	Config() *Config

	Queryer

	// TransactionInfo returns the transaction info of the connection
	TransactionInfo() TransactionInfo

	// Begin a new transaction.
	// If the connection is already a transaction then a brand
	// new transaction will begin based on the connection
	// that started this transaction.
	// The passed no will be returnd from the transaction's
	// Connection.TransactionNo method.
	// Implementations should use the function NextTransactionNo
	// to aquire a new number in a threadsafe way.
	Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (Connection, error)

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
