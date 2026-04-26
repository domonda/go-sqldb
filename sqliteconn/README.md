# sqliteconn

SQLite connection implementation for [github.com/domonda/go-sqldb](https://github.com/domonda/go-sqldb) using [zombiezen.com/go/sqlite](https://zombiezen.com/go/sqlite).

This package provides a native wrapper around zombiezen.com/go/sqlite that implements the `sqldb.Connection` interface without using the `database/sql` package.

## Features

- Implements `sqldb.Connection` interface
- `QueryBuilder` implements `sqldb.QueryBuilder`, `sqldb.UpsertQueryBuilder`, and `sqldb.ReturningQueryBuilder`
- Automatic foreign key constraint enforcement
- WAL mode enabled by default for better concurrency
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
- `PRAGMA query_only = ON` - For read-only connections

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

### Limitations

- **No LISTEN/NOTIFY**: SQLite does not support PostgreSQL's LISTEN/NOTIFY functionality. Calling these methods will return an error.
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
