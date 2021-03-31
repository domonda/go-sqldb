package db

import (
	"context"
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
