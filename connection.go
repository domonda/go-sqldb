package sqldb

type (
	OnNotifyFunc   func(channel, payload string)
	OnUnlistenFunc func(channel string)
)

type Connection interface {
	Exec(query string, args ...interface{}) error

	// InsertStruct inserts a new row into table using the exported fields
	// of rowStruct which have a `db` tag that is not "-".
	// If optional onlyColumns are provided, then only struct fields with a `db` tag
	// matching any of the passed column names will be inserted.
	InsertStruct(table string, rowStruct interface{}, onlyColumns ...string) error

	QueryRow(query string, args ...interface{}) RowScanner
	QueryRows(query string, args ...interface{}) RowsScanner

	Begin() (Connection, error)
	Commit() error
	Rollback() error
	Transaction(txFunc func(tx Connection) error) error

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
