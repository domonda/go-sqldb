/*
Package pqconn implements github.com/domonda/go-sqldb.Connection
for PostgreSQL using github.com/lib/pq.

Basic usage:

	import (
		"context"
		"github.com/domonda/go-sqldb"
		"github.com/domonda/go-sqldb/pqconn"
	)

	config := &sqldb.ConnConfig{
		Driver:   pqconn.Driver,
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "mydb",
		Extra:    map[string]string{"sslmode": "disable"},
	}

	conn, err := pqconn.Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

The connection uses $1, $2, ... placeholders and PostgreSQL identifier quoting.

PostgreSQL-specific features:
  - LISTEN/NOTIFY via ListenOnChannel, UnlistenChannel, and IsListeningOnChannel
  - Read-only mode (sets default_transaction_read_only = on)
  - Typed error inspection (IsUniqueViolation, IsForeignKeyViolation, etc.)
  - Default isolation level is sql.LevelReadCommitted
*/
package pqconn
