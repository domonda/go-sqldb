# mssqlconn

Package `mssqlconn` implements the `github.com/domonda/go-sqldb` interfaces for Microsoft SQL Server using the [`github.com/microsoft/go-mssqldb`](https://github.com/microsoft/go-mssqldb) driver.

## Connecting

Use `Connect` to establish a connection from an `sqldb.ConnConfig`:

```go
config := &sqldb.ConnConfig{
    Driver:   mssqlconn.Driver, // "sqlserver"
    Host:     "localhost",
    Port:     1433,
    User:     "sa",
    Password: "secret",
    Database: "mydb",
}
conn, err := mssqlconn.Connect(ctx, config)
```

`MustConnect` panics on error.

Extra connection parameters can be passed via `config.Extra` and are appended as URL query parameters to the connection string:

```go
config.Extra = map[string]string{
    "encrypt": "disable",
}
```

## Query Builder

`QueryBuilder` implements `sqldb.QueryBuilder` and `sqldb.UpsertQueryBuilder`:

- Standard CRUD via embedded `sqldb.StdQueryBuilder`
- Upsert via `MERGE INTO ... USING ... WHEN MATCHED THEN UPDATE ... WHEN NOT MATCHED THEN INSERT`
- `InsertUnique` uses `MERGE` with `WHEN NOT MATCHED THEN INSERT` (rows affected indicates whether a row was inserted)

It does not implement `sqldb.ReturningQueryBuilder`.

## Query Formatting

`QueryFormatter` implements `sqldb.QueryFormatter` with SQL Server-specific formatting:

- Table and column names are escaped using bracket quoting (`[identifier]`)
- Placeholders use `@p1`, `@p2`, ... syntax
- `EscapeIdentifier` quotes identifiers that contain special characters or are T-SQL reserved words
- Maximum number of arguments per query: 2100

## Identifier Escaping

`EscapeIdentifier` wraps identifiers in brackets when necessary:

```go
mssqlconn.EscapeIdentifier("order")   // → [order] (reserved word)
mssqlconn.EscapeIdentifier("name")    // → name    (safe, no quoting)
mssqlconn.EscapeIdentifier("my col")  // → [my col] (contains space)
```

Schema-qualified table names (`schema.table`) are supported by `QueryFormatter.FormatTableName`.

## Error Inspection

SQL Server-specific constraint errors are automatically wrapped with the corresponding `sqldb` error types, so they can be checked with `errors.As` or `errors.Is`.

Helper functions are also available for direct inspection:

| Function                              | SQL Server Error | Description                              |
| ------------------------------------- | ---------------- | ---------------------------------------- |
| `IsNotNullViolation(err)`             |              515 | NULL inserted into NOT NULL column       |
| `IsUniqueViolation(err)`              |       2601, 2627 | Duplicate key (unique index or constraint) |
| `IsForeignKeyViolation(err, ...)`     |              547 | Foreign key constraint violated          |
| `IsCheckViolation(err)`               |              547 | CHECK constraint violated                |
| `IsDeadlockDetected(err)`             |             1205 | Transaction deadlock detected            |

```go
err := db.Exec(ctx, "INSERT INTO orders ...")
if mssqlconn.IsUniqueViolation(err) {
    // handle duplicate key
}
if mssqlconn.IsForeignKeyViolation(err) {
    // handle FK violation
}
```

### Generic `sqldb` errors

- [ ] `ErrIntegrityConstraintViolation`
- [x] `ErrNotNullViolation`
- [x] `ErrUniqueViolation`
- [x] `ErrForeignKeyViolation`
- [x] `ErrCheckViolation`
- [ ] `ErrRestrictViolation`
- [ ] `ErrExclusionViolation`
- [x] `ErrDeadlock`
- [x] `ErrRaisedException`
- [ ] `ErrQueryCanceled`
- [ ] `ErrNullValueNotAllowed`

These are wrapped automatically and can be inspected with `errors.As`:

```go
var uniqueErr sqldb.ErrUniqueViolation
if errors.As(err, &uniqueErr) {
    fmt.Println("violated constraint:", uniqueErr.Constraint)
}
```

## Drop Schema for Testing

Three functions are provided for resetting the database to a clean state in tests.

| Function          | Drops                                    |
| ----------------- | ---------------------------------------- |
| `DropAllTables`   | All user tables (removes FK constraints first) |
| `DropAllTypes`    | All user-defined types                   |
| `DropAll`         | Tables then types (correct order)        |

Always use `DropAll` (or call `DropAllTables` before `DropAllTypes`) because types referenced by tables cannot be dropped while the tables exist.

### Example: Test Setup

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    config := &sqldb.ConnConfig{
        Driver:   mssqlconn.Driver,
        Host:     "localhost",
        Port:     1433,
        User:     "sa",
        Password: "secret",
        Database: "myapp_test",
    }
    conn, err := mssqlconn.Connect(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    db.SetConn(conn)

    // Drop everything and recreate schema
    err = mssqlconn.DropAll(ctx, conn)
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
