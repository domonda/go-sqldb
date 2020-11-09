package sqldb

import (
	"context"
)

type (
	OnNotifyFunc   func(channel, payload string)
	OnUnlistenFunc func(channel string)
)

// Exec executes a query with optional args.
func Exec(ctx context.Context, query string, args ...interface{}) error {
	if tx := ctx.Value(txKey); tx != nil {
		return tx.(Tx).Exec(ctx, query, args)
	}
	conn, err := Conn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, query, args)
}

// Insert a new row into table using the values.
func Insert(ctx context.Context, table string, values Values) error {
	panic("todo")
}

// InsertUnique inserts a new row into table using the passed values
// or does nothing if the onConflict statement applies.
// Returns if a row was inserted.
func InsertUnique(ctx context.Context, table string, values Values, onConflict string) (inserted bool, err error) {
	panic("todo")
}

// InsertReturning inserts a new row into table using values
// and returns values from the inserted row listed in returning.
func InsertReturning(ctx context.Context, table string, values Values, returning string) RowScanner {
	panic("todo")
}

// InsertStruct inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
func InsertStruct(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	panic("todo")
}

// InsertStructIgnoreColumns inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
func InsertStructIgnoreColumns(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	panic("todo")
}

// InsertUniqueStruct inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// Does nothing if the onConflict statement applies and returns if a row was inserted.
func InsertUniqueStruct(ctx context.Context, table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	panic("todo")
}

// InsertUniqueStructIgnoreColumns inserts a new row into table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// Does nothing if the onConflict statement applies and returns if a row was inserted.
func InsertUniqueStructIgnoreColumns(ctx context.Context, table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	panic("todo")
}

// Update table rows(s) with values using the where statement with passed in args starting at $1.
func Update(ctx context.Context, table string, values Values, where string, args ...interface{}) error {
	panic("todo")
}

// UpdateReturningRow updates a table row with values using the where statement with passed in args starting at $1
// and returning a single row with the columns specified in returning argument.
func UpdateReturningRow(ctx context.Context, table string, values Values, returning, where string, args ...interface{}) RowScanner {
	panic("todo")
}

// UpdateReturningRows updates table rows with values using the where statement with passed in args starting at $1
// and returning multiple rows with the columns specified in returning argument.
func UpdateReturningRows(ctx context.Context, table string, values Values, returning, where string, args ...interface{}) RowsScanner {
	panic("todo")
}

// UpdateStruct updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
func UpdateStruct(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	panic("todo")
}

// UpdateStructIgnoreColumns updates a row in a table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
func UpdateStructIgnoreColumns(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	panic("todo")
}

// UpsertStruct upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// If restrictToColumns are provided, then only struct fields with a `db` tag
// matching any of the passed column names will be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStruct(ctx context.Context, table string, rowStruct interface{}, restrictToColumns ...string) error {
	panic("todo")
}

// UpsertStructIgnoreColumns upserts a row to table using the exported fields
// of rowStruct which have a `db` tag that is not "-".
// Struct fields with a `db` tag matching any of the passed ignoreColumns will not be used.
// The struct must have at least one field with a `db` tag value having a ",pk" suffix
// to mark primary key column(s).
// If inserting conflicts on the primary key column(s), then an update is performed.
func UpsertStructIgnoreColumns(ctx context.Context, table string, rowStruct interface{}, ignoreColumns ...string) error {
	panic("todo")
}

// QueryRow queries a single row and returns a RowScanner for the results.
func QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner {
	panic("todo")
}

// QueryRows queries multiple rows and returns a RowsScanner for the results.
func QueryRows(ctx context.Context, query string, args ...interface{}) RowsScanner {
	var (
		rows Rows
		err  error
	)
	if tx := ctx.Value(txKey); tx != nil {
		rows, err = tx.(Tx).Query(ctx, query, args)
	} else {
		conn, e := Conn(ctx)
		if e != nil {
			return RowsScannerWithError(e)
		}
		rows, err = conn.Query(ctx, query, args)
	}
	if err != nil {
		return RowsScannerWithError(err)
	}
	return newRowsScanner(ctx, rows, TODO_StructFieldNamer, query, args)
}

// IsTransaction returns if the context has a transaction
func IsTransaction(ctx context.Context) bool {
	return ctx.Value(txKey) != nil
}

// ListenOnChannel will call onNotify for every channel notification
// and onUnlisten if the channel gets unlistened
// or the listener connection gets closed for some reason.
// It is valid to pass nil for onNotify or onUnlisten to not get those callbacks.
// Note that the callbacks are called in sequence from a single go routine,
// so callbacks should offload long running or potentially blocking code to other go routines.
// Panics from callbacks will be recovered and logged.
func ListenOnChannel(ctx context.Context, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	panic("todo")
}

// UnlistenChannel will stop listening on the channel.
// An error is returned, when the channel was not listened to
// or the listener connection is closed.
func UnlistenChannel(ctx context.Context, channel string) error {
	panic("todo")
}

// IsListeningOnChannel returns if a channel is listened to.
func IsListeningOnChannel(ctx context.Context, channel string) bool {
	panic("todo")
}
