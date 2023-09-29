package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// New creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != "mysql" {
		return nil, fmt.Errorf(`invalid driver %q, mysqlconn expects "mysql"`, config.Driver)
	}
	config.DefaultIsolationLevel = sql.LevelRepeatableRead // mysql default

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return impl.NewGenericConnection(
		ctx,
		db,
		config,
		sqldb.UnsupportedListener(),
		sqldb.DefaultStructFieldMapping,
		validateColumnName,
		nil, // No driver.ValueConverter necessary because MySQL doesn't suport arrays
		argFmt,
	), nil
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
