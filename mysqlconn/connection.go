package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// Connect creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context and only returned
// and only returned when there was no error from the ping.
func Connect(ctx context.Context, config *sqldb.ConnConfig) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return sqldb.NewGenericConn(db, config, sql.LevelRepeatableRead), nil
}

// MustConnect creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context and only returned
// and only returned when there was no error from the ping.
// Errors are paniced.
func MustConnect(ctx context.Context, config *sqldb.ConnConfig) sqldb.Connection {
	conn, err := Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}
