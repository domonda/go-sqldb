package sqliteconn

import (
	"context"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/domonda/go-sqldb"
)

// New creates a new sqldb.Connection using the passed sqldb.Config
// and modernc.org/sqlite as driver implementation.
// The connection is pinged with the passed context
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != "sqlite" {
		return nil, fmt.Errorf(`invalid driver %q, pqconn expects "sqlite"`, config.Driver)
	}

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	_ = db
	panic("TODO")
}
