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

`ConnectExt` wraps `Connect` and returns an `sqldb.ConnExt` with a struct reflector, SQL Server query formatter, and query builder:

```go
conn, err := mssqlconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
```

`MustConnect` and `MustConnectExt` panic on error.

Extra connection parameters can be passed via `config.Extra` and are appended as URL query parameters to the connection string:

```go
config.Extra = map[string]string{
    "encrypt": "disable",
}
```

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
    conn, err := mssqlconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
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
