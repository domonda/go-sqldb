/*
Package mssqlconn implements github.com/domonda/go-sqldb.Connection
for Microsoft SQL Server using github.com/microsoft/go-mssqldb.

Basic usage:

	import (
		"context"
		"github.com/domonda/go-sqldb"
		"github.com/domonda/go-sqldb/mssqlconn"
	)

	config := &sqldb.Config{
		Driver:   mssqlconn.Driver,
		Host:     "localhost",
		Port:     1433,
		User:     "sa",
		Password: "secret",
		Database: "mydb",
	}

	conn, err := mssqlconn.Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

The connection uses @p1, @p2, ... placeholders and bracket identifier quoting ([identifier]).

SQL Server-specific features:
  - Default isolation level is sql.LevelReadCommitted
  - EscapeIdentifier wraps identifiers in brackets when needed
  - DropAll, DropAllTables, and DropAllTypes for resetting test databases
*/
package mssqlconn
