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

	config := &sqldb.Config{
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
  - Sets PRAGMA busy_timeout = DefaultBusyTimeoutMs (5000 ms) — override via
    Config.Extra["busy_timeout"] as a non-negative millisecond integer; "0"
    restores SQLite's native fail-fast behavior
  - Sets query_only mode when Config.ReadOnly is true

Multi-process access:

The same database file may be opened from multiple processes. Inter-process
coordination uses the standard SQLite VFS file-locking primitives (fcntl on
Unix, LockFileEx on Windows). With WAL enabled, readers and writers do not
block each other; only writers serialize against each other across processes.
busy_timeout makes contended writes wait instead of failing immediately.

Caveats: WAL is not safe on NFS or other network filesystems — use a local
filesystem. Begin uses BEGIN DEFERRED unless sql.TxOptions.Isolation is
>= sql.LevelReadCommitted; for write-heavy multi-process workloads, request
that isolation level so the write lock is acquired up front via BEGIN
IMMEDIATE. Errors from exhausted busy_timeout match sqliteconn.IsDatabaseLocked.
See sqliteconn/README.md for details.

SQLite-specific features:
  - Default isolation level is sql.LevelSerializable
  - Uses ?1, ?2, ... positional placeholders (SQLite numbered parameters)
  - Does not support LISTEN/NOTIFY (PostgreSQL-specific feature)
  - All constraint violations are wrapped with appropriate sqldb.Err* types

For in-memory databases, use ":memory:" as the database name:

	config := &sqldb.Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}
*/
package sqliteconn
