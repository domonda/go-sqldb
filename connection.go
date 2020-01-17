package sqldb

import (
	"context"
	"database/sql"
)

type (
	OnNotifyFunc   func(channel, payload string)
	OnUnlistenFunc func(channel string)
)

// Values is a map from column names to values
type Values map[string]interface{}

// Connection represents a database connection or transaction
type Connection interface {
	// WithStructFieldNamer returns a copy of the connection
	// that will use the passed StructFieldNamer.
	WithStructFieldNamer(namer StructFieldNamer) Connection
	StructFieldNamer() StructFieldNamer

	Ping(ctx context.Context) error

	// Exec executes a query with optional args.
	Exec(query string, args ...interface{}) error

	// ExecContext executes a query with optional args.
	ExecContext(ctx context.Context, query string, args ...interface{}) error

	// Insert a new row into table using the values.
	Insert(table string, values Values) error

	// InsertContext inserts a new row into table using the values.
	InsertContext(ctx context.Context, table string, values Values) error

	// InsertReturning inserts a new row into table using values
	// and returns values from the inserted row listed in returning.
	InsertReturning(table string, values Values, returning string) RowScanner

	// InsertReturningContext inserts a new row into table using values
	// and returns values from the inserted row listed in returning.
	InsertReturningContext(ctx context.Context, table string, values Values, returning string) RowScanner

	// InsertStruct inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error

	// InsertStructContext inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	InsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error

	// InsertStructIgnoreColums inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	InsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error

	// InsertStructIgnoreColumsContext inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	InsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error

	// UpsertStruct upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// If inserting conflicts on idColumn, then an update of the existing row is performed.
	UpsertStruct(table string, rowStruct interface{}, idColumn string, restrictToColumns ...string) error

	// UpsertStructContext upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// If inserting conflicts on idColumn, then an update of the existing row is performed.
	UpsertStructContext(ctx context.Context, table string, rowStruct interface{}, idColumn string, restrictToColumns ...string) error

	// UpsertStructIgnoreColums upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// If inserting conflicts on idColumn, then an update of the existing row is performed.
	UpsertStructIgnoreColums(table string, rowStruct interface{}, idColumn string, ignoreColumns ...string) error

	// UpsertStructIgnoreColumsContext upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// If inserting conflicts on idColumn, then an update of the existing row is performed.
	UpsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, idColumn string, ignoreColumns ...string) error

	// QueryRow queries a single row and returns a RowScanner for the results.
	QueryRow(query string, args ...interface{}) RowScanner

	// QueryRowContext queries a single row and returns a RowScanner for the results.
	QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner

	// QueryRows queries multiple rows and returns a RowsScanner for the results.
	QueryRows(query string, args ...interface{}) RowsScanner

	// QueryRowsContext queries multiple rows and returns a RowsScanner for the results.
	QueryRowsContext(ctx context.Context, query string, args ...interface{}) RowsScanner

	Begin(ctx context.Context, opts *sql.TxOptions) (Connection, error)
	Commit() error
	Rollback() error

	// Transaction executes txFunc within a database transaction that is passed in as tx Connection.
	// The transaction will be rolled back if txFunc returns an error or panics.
	// Recovered panics are re-paniced after the transaction was rolled back.
	// Transaction returns errors from txFunc or transaction commit errors happening after txFunc.
	// Rollback errors are logged with sqldb.ErrLogger.
	Transaction(ctx context.Context, opts *sql.TxOptions, txFunc func(tx Connection) error) error

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

	Close() error
}

type RowScanner interface {
	Scan(dest ...interface{}) error
	ScanStruct(dest interface{}) error
}

type RowsScanner interface {
	ScanSlice(dest interface{}) error
	ScanStructSlice(dest interface{}) error
	ForEachRow(callback func(RowScanner) error) error
}
