package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// New creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}
	config.DefaultIsolationLevel = sql.LevelRepeatableRead // mysql default

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return sqldb.NewGenericConn(db, config, sqldb.StdQueryFormatter{}), nil
}

// MustNew creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
// Errors are paniced.
func MustNew(ctx context.Context, config *sqldb.Config) sqldb.Connection {
	conn, err := New(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}
