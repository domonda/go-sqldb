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

// NewConnExt creates a new sqldb.ConnExt with MSSQL-specific components.
// It combines the passed connection and struct reflector with MySQL
// specific QueryFormatter and QueryBuilder.
func NewConnExt(conn sqldb.Connection, structReflector sqldb.StructReflector) *sqldb.ConnExt {
	return sqldb.NewConnExt(
		conn,
		structReflector,
		QueryFormatter{},
		sqldb.StdQueryBuilder{},
	)
}
