package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/domonda/go-sqldb"
)

// ReplaceErrNoRows returns the passed replacement error
// if errors.Is(err, sql.ErrNoRows),
// else err is returned unchanged.
func ReplaceErrNoRows(err, replacement error) error {
	return sqldb.ReplaceErrNoRows(err, replacement)
}

// Now returns the result of the SQL now() function
// using the sqldb.Connection from the passed context.
// This is useful to get the timestamp of a
// SQL transaction for use in Go code.
func Now(ctx context.Context) (time.Time, error) {
	var now time.Time
	err := Conn(ctx).QueryRow(`select now()`).Scan(&now)
	if err != nil {
		return time.Time{}, err
	}
	return now, nil
}

// DebugPrintConn prints a line to stderr using the passed args
// and appending the transaction state of the connection
// and the current time of the database using `select now()`
// or an error if the time could not be queried.
func DebugPrintConn(ctx context.Context, args ...interface{}) {
	opts, isTx := Conn(ctx).TransactionOptions()
	if isTx {
		args = append(args, "SQL-Transaction")
		if optsStr := TxOptionsString(opts); optsStr != "" {
			args = append(args, "Isolation", optsStr)
		}
	}
	now, err := Now(ctx)
	if err == nil {
		args = append(args, "NOW():", now)
	} else {
		args = append(args, "ERROR:", err)
	}
	fmt.Fprintln(os.Stderr, args...)
}

// TxOptionsString returns a string representing the
// passed TxOptions wich will be empty for the default options.
func TxOptionsString(opts *sql.TxOptions) string {
	switch {
	case opts == nil:
		return ""
	case opts.ReadOnly && opts.Isolation == sql.LevelDefault:
		return "Read-Only"
	case opts.ReadOnly && opts.Isolation != sql.LevelDefault:
		return "Read-Only " + opts.Isolation.String()
	case opts.Isolation != sql.LevelDefault:
		return opts.Isolation.String()
	default:
		return ""
	}
}
