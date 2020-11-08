package sqldb

import (
	"context"
	"database/sql"
)

type (
	OnNotifyFunc   func(channel, payload string)
	OnUnlistenFunc func(channel string)
)

// Connection represents a database connection or transaction
type Connection interface {
	// WithContext returns a connection that uses the passed
	// context for its operations.
	WithContext(ctx context.Context) Connection

	// WithStructFieldNamer returns a copy of the connection
	// that will use the passed StructFieldNamer.
	WithStructFieldNamer(namer StructFieldNamer) Connection

	// StructFieldNamer used by methods of this Connection.
	StructFieldNamer() StructFieldNamer

	// Ping returns an error if the database
	// does not answer on this connection.
	Ping() error

	// Stats returns the sql.DBStats of this connection.
	Stats() sql.DBStats

	// Config returns the configuration used
	// to create this connection.
	Config() *Config

	// Exec executes a query with optional args.
	Exec(query string, args ...interface{}) error

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
	InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error

	// InsertStructIgnoreColumns inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error

	// InsertUniqueStruct inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// Does nothing if the onConflict statement applies and returns if a row was inserted.
	InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error)

	// InsertUniqueStructIgnoreColumns inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// Does nothing if the onConflict statement applies and returns if a row was inserted.
	InsertUniqueStructIgnoreColumns(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error)

	// Update table rows(s) with values using the where statement with passed in args starting at $1.
	Update(table string, values Values, where string, args ...interface{}) error

	// UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
	// and returning a single row with the columns specified in returning argument.
	UpdateReturningRow(table string, values Values, returning, where string, args ...interface{}) RowScanner

	// UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
	// and returning multiple rows with the columns specified in returning argument.
	UpdateReturningRows(table string, values Values, returning, where string, args ...interface{}) RowsScanner

	// UpdateStruct updates a row in a table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error

	// UpdateStructIgnoreColumns updates a row in a table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error

	// UpsertStruct upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error

	// UpsertStructIgnoreColumns upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error

	// QueryRow queries a single row and returns a RowScanner for the results.
	QueryRow(query string, args ...interface{}) RowScanner

	// QueryRows queries multiple rows and returns a RowsScanner for the results.
	QueryRows(query string, args ...interface{}) RowsScanner

	// IsTransaction returns if the connection is a transaction
	IsTransaction() bool

	// Begin a new transaction.
	// Returns ErrWithinTransaction if the connection
	// is already within a transaction.
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
