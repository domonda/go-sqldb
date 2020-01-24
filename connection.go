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
	Stats() sql.DBStats
	Config() *Config

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

	// UpdateStruct updates a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error

	// UpdateStructContext updates a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error

	// UpdateStructIgnoreColums updates a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error

	// UpdateStructIgnoreColumsContext updates a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	UpdateStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error

	// UpsertStruct upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error

	// UpsertStructContext upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If restrictToColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStructContext(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error

	// UpsertStructIgnoreColums upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStructIgnoreColums(table string, rowStruct interface{}, ignoreColumns ...string) error

	// UpsertStructIgnoreColumsContext upserts a row to table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
	// The struct must have at least one field with a `db` tag value having a ",pk" suffix
	// to mark primary key column(s).
	// If inserting conflicts on the primary key column(s), then an update is performed.
	UpsertStructIgnoreColumsContext(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error

	// QueryRow queries a single row and returns a RowScanner for the results.
	QueryRow(query string, args ...interface{}) RowScanner

	// QueryRowContext queries a single row and returns a RowScanner for the results.
	QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner

	// QueryRows queries multiple rows and returns a RowsScanner for the results.
	QueryRows(query string, args ...interface{}) RowsScanner

	// QueryRowsContext queries multiple rows and returns a RowsScanner for the results.
	QueryRowsContext(ctx context.Context, query string, args ...interface{}) RowsScanner

	// IsTransaction returns if the connection is a transaction
	IsTransaction() bool
	Begin(ctx context.Context, opts *sql.TxOptions) (Connection, error)
	Commit() error
	Rollback() error

	// Transaction executes txFunc within a database transaction.
	// The transaction will be rolled back if txFunc returns an error or panics.
	// Recovered panics are re-paniced after the transaction is rolled back.
	// Rollback errors are logged with sqldb.ErrLogger.
	// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
	// If this connection is already a transaction, then txFunc is executed within this transaction
	// ignoring opts and without calling another Begin or Commit in this Transaction call.
	// Errors or panics will roll back the inherited transaction though.
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
	// ScanSlice scans one value per row into one slice element of dest.
	// dest must be a pointer to a slice with a row value compatible element type.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// It is an error to query more than one column.
	ScanSlice(dest interface{}) error

	// ScanStructSlice scans every row into the struct fields of dest slice elements.
	// dest must be a pointer to a slice of structs or struct pointers.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// Every mapped struct field must have a corresponding column in the query results.
	ScanStructSlice(dest interface{}) error

	// ForEachRow will call the passed callback with a RowScanner for every row.
	// In case of zero rows, no error will be returned.
	ForEachRow(callback func(RowScanner) error) error
}
