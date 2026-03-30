/*
Package sqliteconn implements github.com/domonda/go-sqldb.Connection
using zombiezen.com/go/sqlite natively.

This package wraps zombiezen.com/go/sqlite directly without using database/sql.

Basic usage:

	import (
		"context"
		"github.com/domonda/go-sqldb"
		"github.com/domonda/go-sqldb/sqliteconn"
	)

	config := &sqldb.ConnConfig{
		Driver:   "sqlite",
		Database: "mydb.sqlite",
	}

	conn, err := sqliteconn.Connect(context.Background(), config)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// conn implements sqldb.Connection
	// and can be used directly with the db package:
	db.SetConn(conn)

The connection automatically:
  - Enables foreign key constraints (PRAGMA foreign_keys = ON)
  - Enables WAL mode for better concurrency (PRAGMA journal_mode = WAL)
  - Sets query_only mode when ConnConfig.ReadOnly is true

SQLite-specific features:
  - Default isolation level is sql.LevelSerializable
  - Uses ?1, ?2, ... positional placeholders (SQLite numbered parameters)
  - Does not support LISTEN/NOTIFY (PostgreSQL-specific feature)
  - All constraint violations are wrapped with appropriate sqldb.Err* types

For in-memory databases, use ":memory:" as the database name:

	config := &sqldb.ConnConfig{
		Driver:   "sqlite",
		Database: ":memory:",
	}
*/
package sqliteconn
