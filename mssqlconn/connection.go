package mssqlconn

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/domonda/go-sqldb"
)

const Driver = "sqlserver"

func Connect(ctx context.Context, config *sqldb.ConnConfig) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return sqldb.NewGenericConn(db, config, sql.LevelReadCommitted), nil
}

func MustConnect(ctx context.Context, config *sqldb.ConnConfig) sqldb.Connection {
	conn, err := Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}
