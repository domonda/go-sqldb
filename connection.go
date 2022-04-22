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

	// WithStructFieldNamer returns a copy of the connection
	// that will use the passed StructFieldNamer.
	WithStructFieldNamer(namer StructFieldNamer) Connection

	// StructFieldNamer used by methods of this Connection.
	StructFieldNamer() StructFieldNamer

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

	// Exec executes a query with optional args.
	Exec(query string, args ...any) error

	// Insert a new row into table using the values.
	Insert(table string, values Values) error

	// InsertUnique inserts a new row into table using the passed values
	// or does nothing if the onConflict statement applies.
	// Returns if a row was inserted.
	InsertUnique(table string, values Values, onConflict string) (inserted bool, err error)

	// InsertReturning inserts a new row into table using values
	// and returns values from the inserted row listed in returning.
	InsertReturning(table string, values Values, returning string) RowScanner

	// InsertStruct inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	InsertStruct(table string, rowStruct any, restrictToColumns ...string) error

	// TODO insert multiple structs with single query if possible
	// InsertStructs(table string, rowStructs any, restrictToColumns ...string) error

	// InsertStructIgnoreColumns inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	InsertStructIgnoreColumns(table string, rowStruct any, ignoreColumns ...string) error

	// InsertUniqueStruct inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// Does nothing if the onConflict statement applies and returns if a row was inserted.
	InsertUniqueStruct(table string, rowStruct any, onConflict string, restrictToColumns ...string) (inserted bool, err error)

	// InsertUniqueStructIgnoreColumns inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// Does nothing if the onConflict statement applies and returns if a row was inserted.
	InsertUniqueStructIgnoreColumns(table string, rowStruct any, onConflict string, ignoreColumns ...string) (inserted bool, err error)

	// Update table rows(s) with values using the where statement with passed in args starting at $1.
	Update(table string, values Values, where string, args ...any) error

	// UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
	// and returning a single row with the columns specified in returning argument.
	UpdateReturningRow(table string, values Values, returning, where string, args ...any) RowScanner

	// UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
	// and returning multiple rows with the columns specified in returning argument.
	UpdateReturningRows(table string, values Values, returning, where string, args ...any) RowsScanner

	// UpdateStruct updates a row in a table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStruct(table string, rowStruct any, restrictToColumns ...string) error

	// UpdateStructIgnoreColumns updates a row in a table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStructIgnoreColumns(table string, rowStruct any, ignoreColumns ...string) error

	// UpsertStruct upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStruct(table string, rowStruct any, restrictToColumns ...string) error

	// UpsertStructIgnoreColumns upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStructIgnoreColumns(table string, rowStruct any, ignoreColumns ...string) error

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
