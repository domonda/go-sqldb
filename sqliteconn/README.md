# sqliteconn

SQLite connection implementation for [github.com/domonda/go-sqldb](https://github.com/domonda/go-sqldb) using [zombiezen.com/go/sqlite](https://zombiezen.com/go/sqlite).

This package provides a native wrapper around zombiezen.com/go/sqlite that implements the `sqldb.Connection` interface without using the `database/sql` package.

## Features

- Implements `sqldb.Connection` interface
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

config := &sqldb.ConnConfig{
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
config := &sqldb.ConnConfig{
    Driver:   "sqlite",
    Database: ":memory:",
}

conn, err := sqliteconn.Connect(context.Background(), config)
```

### Read-Only Mode

```go
config := &sqldb.ConnConfig{
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

config := &sqldb.ConnConfig{
    Driver:   "sqlite",
    Database: "myapp.db",
}

conn, err := sqliteconn.Connect(context.Background(), config)
if err != nil {
    panic(err)
}

// Create ConnExt with StdQueryFormatter (uses ? placeholders for SQLite)
connExt := sqldb.NewConnExt(
    conn,
    sqldb.NewTaggedStructReflector(),
    sqldb.StdQueryFormatter{}, // Empty PlaceholderPosPrefix uses ? for SQLite
    sqldb.StdQueryBuilder{},
)

db.SetConn(connExt)

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

SQLite constraint violations are automatically wrapped with the appropriate `sqldb.Err*` types:
- `sqldb.ErrForeignKeyViolation`
- `sqldb.ErrUniqueViolation`
- `sqldb.ErrNotNullViolation`
- `sqldb.ErrCheckViolation`

### Limitations

- **No LISTEN/NOTIFY**: SQLite does not support PostgreSQL's LISTEN/NOTIFY functionality. Calling these methods will return an error.
- **Placeholder Syntax**: Use `?` for query placeholders (standard SQLite syntax). When using with `sqldb.StdQueryFormatter{}`, it automatically uses `?` placeholders.

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
