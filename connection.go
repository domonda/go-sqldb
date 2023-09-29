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
	// that will use the passed StructFieldMapper.
	WithStructFieldMapper(StructFieldMapper) Connection

	// StructFieldMapper used by methods of this Connection.
	StructFieldMapper() StructFieldMapper

	// ValidateColumnName returns an error
	// if the passed name is not valid for a
	// column of the connection's database.
	ValidateColumnName(name string) error

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

	// Insert a new row into table using the values.
	Insert(table string, values Values) error

	// InsertUnique inserts a new row into table using the passed values
	// or does nothing if the onConflict statement applies.
	// Returns if a row was inserted.
	InsertUnique(table string, values Values, onConflict string) (inserted bool, err error)

	// InsertReturning inserts a new row into table using values
	// and returns values from the inserted row listed in returning.
	InsertReturning(table string, values Values, returning string) RowScanner

	// InsertStruct inserts a new row into table using the connection's
	// StructFieldMapper to map struct fields to column names.
	// Optional ColumnFilter can be passed to ignore mapped columns.
	InsertStruct(table string, rowStruct any, ignoreColumns ...ColumnFilter) error

	// InsertStructs inserts a slice or array of structs
	// as new rows into table using the connection's
	// StructFieldMapper to map struct fields to column names.
	// Optional ColumnFilter can be passed to ignore mapped columns.
	InsertStructs(table string, rowStructs any, ignoreColumns ...ColumnFilter) error

	// InsertUniqueStruct inserts a new row into table using the connection's
	// StructFieldMapper to map struct fields to column names.
	// Optional ColumnFilter can be passed to ignore mapped columns.
	// Does nothing if the onConflict statement applies
	// and returns if a row was inserted.
	InsertUniqueStruct(table string, rowStruct any, onConflict string, ignoreColumns ...ColumnFilter) (inserted bool, err error)

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
	UpdateStruct(table string, rowStruct any, ignoreColumns ...ColumnFilter) error

	// UpsertStruct upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStruct(table string, rowStruct any, ignoreColumns ...ColumnFilter) error

	// IsTransaction returns if the connection is a transaction
	IsTransaction() bool

	// TransactionNo returns the globally unique number of the transaction
	// or zero if the connection is not a transaction.
	// Implementations should use the package function NextTransactionNo
	// to aquire a new number in a threadsafe way.
	TransactionNo() uint64

	// TransactionOptions returns the sql.TxOptions of the
	// current transaction and true as second result value,
	// or false if the connection is not a transaction.
	TransactionOptions() (*sql.TxOptions, bool)

	// Begin a new transaction.
	// If the connection is already a transaction then a brand
	// new transaction will begin on the parent's connection.
	// The passed no will be returnd from the transaction's
	// Connection.TransactionNo method.
	// Implementations should use the package function NextTransactionNo
	// to aquire a new number in a threadsafe way.
	Begin(opts *sql.TxOptions, no uint64) (Connection, error)

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
