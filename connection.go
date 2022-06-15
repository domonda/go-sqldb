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
	// Context that all connection operations use.
	// See also WithContext.
	Context() context.Context

	// WithContext returns a connection that uses the passed
	// context for its operations.
	WithContext(ctx context.Context) Connection

	// WithStructFieldMapper returns a copy of the connection
	// that will use the passed StructFieldNamer.
	WithStructFieldMapper(StructFieldMapper) Connection

	// StructFieldMapper used by methods of this Connection.
	StructFieldMapper() StructFieldMapper

	// Ping returns an error if the database
	// does not answer on this connection
	// with an optional timeout.
	// The passed timeout has to be greater zero
	// to be considered.
	Ping(timeout time.Duration) error

	// Stats returns the sql.DBStats of this connection.
	Stats() sql.DBStats

	// Config returns the configuration used
	// to create this connection.
	Config() *Config

	// ValidateColumnName returns an error
	// if the passed name is not valid for a
	// column of the connection's database.
	ValidateColumnName(name string) error

	// ArgFmt returns the format for SQL query arguments
	ArgFmt() string

	// Err returns any current error of the connection
	Err() error

	// Now returns the result of the SQL now()
	// function for the current connection.
	// Useful for getting the timestamp of a
	// SQL transaction for use in Go code.
	Now() (time.Time, error)

	// Exec executes a query with optional args.
	Exec(query string, args ...any) error

	// QueryRow queries a single row and returns a RowScanner for the results.
	QueryRow(query string, args ...any) RowScanner

	// QueryRows queries multiple rows and returns a RowsScanner for the results.
	QueryRows(query string, args ...any) RowsScanner

	// IsTransaction returns if the connection is a transaction
	IsTransaction() bool

	// TransactionOptions returns the sql.TxOptions of the
	// current transaction and true as second result value,
	// or false if the connection is not a transaction.
	TransactionOptions() (*sql.TxOptions, bool)

	// Begin a new transaction.
	// If the connection is already a transaction, a brand
	// new transaction will begin on the parent's connection.
	Begin(opts *sql.TxOptions) (Connection, error)

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
