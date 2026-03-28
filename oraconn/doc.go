/*
Package oraconn implements github.com/domonda/go-sqldb.Connection
for Oracle Database using github.com/sijms/go-ora/v2.

Basic usage:

	import (
		"context"
		"github.com/domonda/go-sqldb"
		"github.com/domonda/go-sqldb/oraconn"
	)

	config := &sqldb.ConnConfig{
		Driver:   oraconn.Driver,
		Host:     "localhost",
		Port:     1521,
		User:     "myuser",
		Password: "secret",
		Database: "FREEPDB1",
	}

	conn, err := oraconn.Connect(ctx, config, true)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

The lowercaseColumns parameter controls whether column names returned by
[sqldb.Rows.Columns] are lowercased. Oracle returns uppercase names for
unquoted identifiers, but Go struct tags conventionally use lowercase.
Set to true when using struct scanning with lowercase db tags.
Oracle SQL itself is case-insensitive for unquoted identifiers,
so this only affects the Go-side column name matching.

The connection uses :1, :2, ... placeholders and double-quote identifier quoting.

Oracle-specific features:
  - Optional lowercasing of column names for struct tag matching
  - Default isolation level is sql.LevelReadCommitted
  - EscapeIdentifier wraps identifiers in double quotes when needed
  - MERGE-based upsert via UpsertQueryBuilder
  - DropAll, DropAllTables, and DropAllTypes for resetting test databases
  - Typed error inspection (IsUniqueViolation, IsForeignKeyViolation, etc.)
*/
package oraconn
