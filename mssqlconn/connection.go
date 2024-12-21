package mssqlconn

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/domonda/go-sqldb"
)

func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}
	config.DefaultIsolationLevel = sql.LevelReadCommitted // Microsoft SQL Server default

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return sqldb.NewGenericConn(db, config, validateColumnName, argFmt), nil
}

func MustNew(ctx context.Context, config *sqldb.Config) sqldb.Connection {
	conn, err := New(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}
