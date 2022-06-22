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

// Connection represents a database connection or transaction
type Connection interface {
	// Config returns the configuration used
	// to create this connection.
	Config() *Config

	// Stats returns the sql.DBStats of this connection.
	Stats() sql.DBStats

	// Ping returns an error if the database
	// does not answer on this connection
	// with an optional timeout.
	// The passed timeout has to be greater zero
	// to be considered.
	Ping(ctx context.Context, timeout time.Duration) error

	// Err returns any current error of the connection
	Err() error

	// Exec executes a query with optional args.
	Exec(ctx context.Context, query string, args ...any) error

	// QueryRow queries a single row and returns a RowScanner for the results.
	QueryRow(ctx context.Context, query string, args ...any) Row

	// QueryRows queries multiple rows and returns a RowsScanner for the results.
	QueryRows(ctx context.Context, query string, args ...any) Rows

	// IsTransaction returns if the connection is a transaction
	IsTransaction() bool

	// TxOptions returns the sql.TxOptions of the
	// current transaction which can be nil for the default options.
	// Use IsTransaction to check if the connection is a transaction.
	TxOptions() *sql.TxOptions

	// Begin a new transaction.
	// If the connection is already a transaction, a brand
	// new transaction will begin on the parent's connection.
	Begin(ctx context.Context, opts *sql.TxOptions) (Connection, error)

	// Commit the current transaction.
	// Returns ErrNotWithinTransaction if the connection
	// is not within a transaction.
	Commit() error

	// Rollback the current transaction.
	// Returns ErrNotWithinTransaction if the connection
	// is not within a transaction.
	Rollback() error

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

	// Close the connection.
	// Transactions will be rolled back.
	Close() error
}
