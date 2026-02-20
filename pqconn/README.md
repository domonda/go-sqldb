# pqconn

Package `pqconn` implements the `github.com/domonda/go-sqldb` interfaces for PostgreSQL using the [`github.com/lib/pq`](https://github.com/lib/pq) driver.

## Connecting

Use `Connect` to establish a connection from an `sqldb.ConnConfig`:

```go
config := &sqldb.ConnConfig{
    Driver:   pqconn.Driver, // "postgres"
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "postgres",
    Database: "mydb",
    Extra:    map[string]string{"sslmode": "disable"},
}
conn, err := pqconn.Connect(ctx, config)
```

`ConnectExt` wraps `Connect` and returns an `sqldb.ConnExt` with a struct reflector, PostgreSQL query formatter, and query builder:

```go
conn, err := pqconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
```

`MustConnect` and `MustConnectExt` panic on error.

## Read-Only Connections

Set `config.ReadOnly = true` to open a connection with `default_transaction_read_only = on`. The connection verifies that read-only mode is active before returning.

## LISTEN/NOTIFY

The connection supports PostgreSQL `LISTEN`/`NOTIFY` via `ListenOnChannel`, `UnlistenChannel`, and `IsListeningOnChannel`. Listeners are shared per connection URL and automatically reconnect.

## Error Inspection

PostgreSQL error codes are wrapped into typed `sqldb` errors. Helper functions check specific error classes:

| Function                             | PostgreSQL Code | Description                                  |
| ------------------------------------ | --------------- | -------------------------------------------- |
| `IsInvalidTextRepresentation(err)`   |           22P02 | Invalid input for type (e.g. bad UUID)       |
| `IsStringDataRightTruncation(err)`   |           22001 | Value too long for column type               |
| `IsRestrictViolation(err)`           |           23001 | RESTRICT constraint violated                 |
| `IsNotNullViolation(err)`            |           23502 | NOT NULL constraint violated                 |
| `IsForeignKeyViolation(err, ...)`    |           23503 | Foreign key constraint violated              |
| `IsUniqueViolation(err)`             |           23505 | Unique constraint violated                   |
| `IsCheckViolation(err)`              |           23514 | CHECK constraint violated                    |
| `IsExclusionViolation(err)`          |           23P01 | Exclusion constraint violated                |
| `IsSerializationFailure(err)`        |           40001 | Serialization conflict (retry transaction)   |
| `IsDeadlockDetected(err)`            |           40P01 | Deadlock detected (retry transaction)        |
| `IsInsufficientPrivilege(err)`       |           42501 | Permission denied                            |
| `IsLockNotAvailable(err)`            |           55P03 | Lock not acquired (e.g. FOR UPDATE NOWAIT)   |
| `IsQueryCanceled(err)`               |           57014 | Query canceled (e.g. context cancellation)   |
| `IsRaisedException(err)`             |           P0001 | PL/pgSQL RAISE EXCEPTION                    |
| `GetRaisedException(err)`            |           P0001 | Returns the exception message                |

## Query Formatting

`QueryFormatter` implements `sqldb.QueryFormatter` with PostgreSQL-specific formatting:

- Table and column names are escaped using PostgreSQL identifier quoting rules
- Placeholders use `$1`, `$2`, ... syntax
- `EscapeIdentifier` quotes identifiers that contain special characters or are reserved words
- `NewTypeMapper` returns a Go-to-PostgreSQL type mapper (e.g. `time.Time` -> `timestamptz`, `int64` -> `bigint`)

## Drop Queries for Testing

The package provides pre-built queries for dropping all user-created database objects. These are useful in tests to reset the database to a clean state before recreating the schema.

### Available Constants

| Constant                            | Scope          | Drops          |
| ----------------------------------- | -------------- | -------------- |
| `DropAllTablesQuery`                | All schemas    | Tables         |
| `DropAllTypesQuery`                 | All schemas    | Types          |
| `DropAllQuery`                      | All schemas    | Tables + Types |
| `DropAllTablesInCurrentSchemaQuery` | Current schema | Tables         |
| `DropAllTypesInCurrentSchemaQuery`  | Current schema | Types          |
| `DropAllInCurrentSchemaQuery`       | Current schema | Tables + Types |

"All schemas" means every user schema (excluding `pg_catalog` and `information_schema`). "Current schema" means `current_schema()`, which is usually `public`.

### Execution Order

Tables **must** be dropped before types. This is because PostgreSQL automatically creates a composite type for every table, and `DROP TYPE` cannot remove these composite types (error `2BP01`). Only `DROP TABLE` removes both the table and its composite type.

The combined queries (`DropAllQuery`, `DropAllInCurrentSchemaQuery`) handle this ordering internally.

If you use the table and type queries separately, always execute them in this order:

1. `DropAllTablesQuery` or `DropAllTablesInCurrentSchemaQuery`
2. `DropAllTypesQuery` or `DropAllTypesInCurrentSchemaQuery`

### Example: Test Setup

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    config := &sqldb.ConnConfig{
        Driver:   pqconn.Driver,
        Host:     "localhost",
        Port:     5432,
        User:     "postgres",
        Password: "postgres",
        Database: "myapp_test",
        Extra:    map[string]string{"sslmode": "disable"},
    }
    conn, err := pqconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
    if err != nil {
        log.Fatal(err)
    }
    db.SetConn(conn)

    // Drop everything and recreate schema
    err = db.Exec(ctx, pqconn.DropAllInCurrentSchemaQuery)
    if err != nil {
        log.Fatal(err)
    }
    err = db.Exec(ctx, mySchema)
    if err != nil {
        log.Fatal(err)
    }

    m.Run()
}
```

Or to drop across all schemas:

```go
err = db.Exec(ctx, pqconn.DropAllQuery)
```
