# sqliteconn

SQLite connection implementation for [github.com/domonda/go-sqldb](https://github.com/domonda/go-sqldb) using [zombiezen.com/go/sqlite](https://zombiezen.com/go/sqlite).

This package provides a native wrapper around zombiezen.com/go/sqlite that implements the `sqldb.Connection` interface without using the `database/sql` package.

## Features

- Implements `sqldb.Connection` interface
- `QueryBuilder` implements `sqldb.QueryBuilder`, `sqldb.UpsertQueryBuilder`, and `sqldb.ReturningQueryBuilder`
- Automatic foreign key constraint enforcement
- Safe multi-process access by default: WAL journal mode and a 5-second `busy_timeout` are set on every connect (see [Process Concurrency](#process-concurrency))
- Read-only mode support
- Proper error wrapping for SQLite constraint violations

## Installation

```bash
go get github.com/domonda/go-sqldb/sqliteconn
```

## Usage

### Basic Connection

```go
import (
    "context"
    "github.com/domonda/go-sqldb"
    "github.com/domonda/go-sqldb/sqliteconn"
)

config := &sqldb.Config{
    Driver:   "sqlite",
    Database: "myapp.db",
}

conn, err := sqliteconn.Connect(context.Background(), config)
if err != nil {
    panic(err)
}
defer conn.Close()
```

### In-Memory Database

```go
config := &sqldb.Config{
    Driver:   "sqlite",
    Database: ":memory:",
}

conn, err := sqliteconn.Connect(context.Background(), config)
```

### Read-Only Mode

```go
config := &sqldb.Config{
    Driver:   "sqlite",
    Database: "myapp.db",
    ReadOnly: true,
}

conn, err := sqliteconn.Connect(context.Background(), config)
```

### Using with db package

```go
import (
    "github.com/domonda/go-sqldb/db"
    "github.com/domonda/go-sqldb/sqliteconn"
)

config := &sqldb.Config{
    Driver:   "sqlite",
    Database: "myapp.db",
}

conn, err := sqliteconn.Connect(context.Background(), config)
if err != nil {
    panic(err)
}

db.SetConn(conn)

// Now use db package functions
err = db.Exec(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
```

## SQLite-Specific Considerations

### Isolation Levels

SQLite's default isolation level is serializable. The connection returns `sql.LevelSerializable` as the default isolation level.

### PRAGMA Settings

The connection automatically sets:
- `PRAGMA foreign_keys = ON` - Enables foreign key constraints
- `PRAGMA journal_mode = WAL` - Enables Write-Ahead Logging for better concurrency
- `PRAGMA busy_timeout = 5000` - Waits up to 5 seconds (`sqliteconn.DefaultBusyTimeoutMs`) on lock contention before returning `SQLITE_BUSY`. Override with `Config.Extra["busy_timeout"]` as a non-negative millisecond integer (`"0"` keeps SQLite's native fail-fast behavior).
- `PRAGMA query_only = ON` - For read-only connections

```go
config := &sqldb.Config{
    Driver:   "sqlite",
    Database: "myapp.db",
    Extra:    map[string]string{"busy_timeout": "10000"}, // wait up to 10s on lock contention
}
```

### Constraint Violations

### Generic `sqldb` errors

- [x] `ErrIntegrityConstraintViolation`
- [x] `ErrNotNullViolation`
- [x] `ErrUniqueViolation`
- [x] `ErrForeignKeyViolation`
- [x] `ErrCheckViolation`
- [ ] `ErrRestrictViolation`
- [ ] `ErrExclusionViolation`
- [ ] `ErrDeadlock`
- [ ] `ErrRaisedException`
- [ ] `ErrQueryCanceled`
- [ ] `ErrNullValueNotAllowed`

### Process Concurrency

The same database file can be opened from multiple OS processes at once. Coordination is handled by SQLite's VFS using OS-level file locks (`fcntl` on Unix, `LockFileEx` on Windows) — `modernc.org/sqlite`, the engine underneath `zombiezen.com/go/sqlite`, uses the standard SQLite VFS, so multi-process semantics match upstream SQLite exactly.

What `Connect` does for you:

- **WAL is on** (`PRAGMA journal_mode = WAL`). Readers do not block writers and writers do not block readers — only writers serialize against each other. The WAL file (`<db>-wal`) and shared-memory file (`<db>-shm`) live next to the database file; all participating processes must have read/write access to the directory.
- **`busy_timeout = 5000` is on**. When two processes contend for the write lock, the loser waits up to 5 seconds for the lock to free instead of returning `SQLITE_BUSY` immediately. Tune via `Config.Extra["busy_timeout"]` (milliseconds). Set `"0"` to restore fail-fast behavior, e.g. in tests.

What you should know:

- **Transaction mode and write contention.** `Begin` issues `BEGIN DEFERRED` by default and upgrades to `BEGIN IMMEDIATE` only when `sql.TxOptions.Isolation >= sql.LevelReadCommitted`. A deferred transaction that starts as a reader and then issues its first `INSERT`/`UPDATE`/`DELETE` may fail with `SQLITE_BUSY_SNAPSHOT` if another process committed in the meantime — and this kind of busy error is **not** retried by `busy_timeout`. For write-heavy multi-process workloads, prefer passing `&sql.TxOptions{Isolation: sql.LevelSerializable}` (or any level `>= sql.LevelReadCommitted`) to take the write lock up front via `BEGIN IMMEDIATE`.
- **Shared-cache is not enabled** (no `OpenSharedCache` flag, no `sqlite3_enable_shared_cache()` call). Shared-cache is a within-process optimization and has no effect on multi-process access regardless of the setting; upstream SQLite recommends WAL over shared-cache anyway.
- **Filesystem requirements.** WAL relies on shared-memory-style coordination (`mmap` of the `-shm` file). It works on local filesystems but is **not safe on NFS or other network filesystems**. Use a local volume or fall back to a non-WAL journal mode on networked storage.
- **One write at a time.** WAL allows many concurrent readers but still only one writer at a time across all processes. If your workload is dominated by writes from multiple processes, expect queueing — `busy_timeout` makes that queueing graceful rather than visible as errors.
- **Detecting timeout exhaustion.** When `busy_timeout` is exhausted (or set to `0`), operations return an error matching `sqliteconn.IsDatabaseLocked`. Use that to decide whether to retry at the application layer.

### Limitations

- **No LISTEN/NOTIFY**: SQLite does not support PostgreSQL's LISTEN/NOTIFY functionality. Calling these methods will return an error.
- **No pinned connections**: `sqliteconn` does not implement `sqldb.ConnPinner`. There is no connection pool — the single underlying connection is already one fixed session — so session-scoped state already persists across calls without pinning. `sqldb.PinConn` returns an error wrapping `errors.ErrUnsupported`, as does `db.PinnedConn` outside a transaction; inside an existing transaction `db.PinnedConn` simply runs the callback unchanged, since the transaction is already bound to one session.
- **Placeholder Syntax**: Uses `?1`, `?2`, ... positional placeholders (SQLite numbered parameters). This is required so that query builders can reference arguments by index regardless of their order in the SQL statement.

## Schema introspection

`sqliteconn` implements `sqldb.Information` using SQLite's native catalog: `sqlite_schema` for object enumeration and `PRAGMA` functions for column / FK / index metadata. SQLite does not expose `information_schema` and has no concept of a schema inside a database — what SQLite calls a "schema" is an attached database (`main`, `temp`, and any `ATTACH DATABASE` names).

| Method            | Source                                                                                                |
| ----------------- | ----------------------------------------------------------------------------------------------------- |
| `Schemas`         | `PRAGMA database_list` — returns attached databases (always `main`, plus `temp` and any `ATTACH`ed names) |
| `CurrentSchema`   | Always `"main"`                                                                                       |
| `Tables`/`Views`  | `<dbname>.sqlite_schema` filtered by `type` (one query per attached database)                         |
| `Columns`         | `PRAGMA table_xinfo(...)` — works for both tables and views                                           |
| `PrimaryKey`      | `PRAGMA table_info(...)` filtered on `pk > 0`, ordered by `pk` (constraint-declaration order)         |
| `ForeignKeys`     | `PRAGMA foreign_key_list(...)` — composite FKs are reassembled from per-column rows by `id`           |
| `Routines`        | `errors.ErrUnsupported` — SQLite has no stored procedures or functions                                |
| `RoutineExists`   | `errors.ErrUnsupported`                                                                               |

**Caveats specific to SQLite:**

- **Synthetic FK names:** `ForeignKeyInfo.Name` is generated as `fk_<id>` (where `<id>` is the value SQLite assigns in `PRAGMA foreign_key_list`). SQLite does not store FK constraint names in its catalog, so there is no real name to return.
- **FK enforcement vs catalog:** the catalog metadata is always available via `PRAGMA foreign_key_list`. `PRAGMA foreign_keys = ON` (set automatically by `Connect`) controls runtime enforcement, not whether `ForeignKeys` returns rows.
- **`OnUpdate`/`OnDelete`:** SQLite reports the action exactly as declared. The driver normalizes it to the standard SQL spellings (`NO ACTION`, `RESTRICT`, `CASCADE`, `SET NULL`, `SET DEFAULT`).

## Error Handling

```go
err := conn.Exec(ctx, "INSERT INTO users (id, name) VALUES (?, ?)", 1, "John")
if sqliteconn.IsUniqueViolation(err) {
    // Handle unique constraint violation
}
if sqliteconn.IsForeignKeyViolation(err) {
    // Handle foreign key violation
}
if sqliteconn.IsDatabaseLocked(err) {
    // Handle database locked error
}
```

## License

MIT
