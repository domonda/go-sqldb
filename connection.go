package sqldb

import (
	"context"
	"database/sql"
)

type (
	OnNotifyFunc   func(channel, payload string)
	OnUnlistenFunc func(channel string)
)

type Values map[string]interface{}

type Connection interface {
	// Exec executes a query with optional args.
	Exec(query string, args ...interface{}) error

	// ExecContext executes a query with optional args.
	ExecContext(ctx context.Context, query string, args ...interface{}) error

	// Insert a new row into table using the columnValues.
	Insert(table string, columnValues Values) error

	// InsertContext inserts a new row into table using the columnValues.
	InsertContext(ctx context.Context, table string, columnValues Values) error

	// InsertReturning inserts a new row into table using columnValues
	// and returns values from the inserted row listed in returning.
	InsertReturning(table string, columnValues Values, returning string) RowScanner

	// InsertReturningContext inserts a new row into table using columnValues
	// and returns values from the inserted row listed in returning.
	InsertReturningContext(ctx context.Context, table string, columnValues Values, returning string) RowScanner

	// InsertStruct inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If optional onlyColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be inserted.
	InsertStruct(table string, rowStruct interface{}, onlyColumns ...string) error

	// InsertStructContext inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If optional onlyColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be inserted.
	InsertStructContext(ctx context.Context, table string, rowStruct interface{}, onlyColumns ...string) error

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
	Transaction(txFunc func(tx Connection) error) error
	TransactionContext(ctx context.Context, opts *sql.TxOptions, txFunc func(tx Connection) error) error

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
