# mysqlconn

Package `mysqlconn` implements the `github.com/domonda/go-sqldb` interfaces for MySQL and MariaDB using the [`github.com/go-sql-driver/mysql`](https://github.com/go-sql-driver/mysql) driver.

## Connecting

Use `Connect` to establish a connection from an `sqldb.ConnConfig`:

```go
config := &sqldb.ConnConfig{
    Driver:   mysqlconn.Driver, // "mysql"
    Host:     "localhost",
    Port:     3306,
    User:     "root",
    Password: "secret",
    Database: "mydb",
}
conn, err := mysqlconn.Connect(ctx, config)
```

`ConnectExt` wraps `Connect` and returns an `sqldb.ConnExt` with a struct reflector, MySQL query formatter, and query builder:

```go
conn, err := mysqlconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
```

`MustConnect` and `MustConnectExt` panic on error.

Extra DSN parameters can be passed via `config.Extra`:

```go
config.Extra = map[string]string{
    "charset":   "utf8mb4",
    "parseTime": "true",
}
```

## Query Formatting

`QueryFormatter` implements `sqldb.QueryFormatter` with MySQL/MariaDB-specific formatting:

- Table and column names are escaped using backtick quoting
- Placeholders use `?` syntax (positional index is ignored; MySQL rebinds by position)
- `EscapeIdentifier` quotes identifiers that contain special characters or are MySQL 8.0 reserved words
- Maximum number of arguments per query: 65535

## Identifier Escaping

`EscapeIdentifier` wraps identifiers in backticks when necessary:

```go
mysqlconn.EscapeIdentifier("order")  // → `order` (reserved word)
mysqlconn.EscapeIdentifier("name")   // → name    (safe, no quoting)
mysqlconn.EscapeIdentifier("my col") // → `my col` (contains space)
```

Schema-qualified table names (`schema.table`) are supported by `QueryFormatter.FormatTableName`.

## Error Inspection

MySQL-specific constraint errors are automatically wrapped with the corresponding `sqldb` error types, so they can be checked with `errors.As` or `errors.Is`.

Helper functions are also available for direct inspection:

| Function                              | MySQL Error | Description                          |
| ------------------------------------- | ----------- | ------------------------------------ |
| `IsNotNullViolation(err)`             |        1048 | NULL inserted into NOT NULL column   |
| `IsUniqueViolation(err)`              |        1062 | Duplicate entry for unique key       |
| `IsForeignKeyViolation(err, ...)`     | 1216/1217/1451/1452 | Foreign key constraint failed |
| `IsCheckViolation(err)`               |        3819 | CHECK constraint violated            |

```go
err := db.Exec(ctx, "INSERT INTO orders ...")
if mysqlconn.IsUniqueViolation(err) {
    // handle duplicate key
}
if mysqlconn.IsForeignKeyViolation(err) {
    // handle FK violation
}
```

Or using the generic `sqldb` error types:

```go
var uniqueErr sqldb.ErrUniqueViolation
if errors.As(err, &uniqueErr) {
    fmt.Println("violated constraint:", uniqueErr.Constraint)
}
```

## Drop Tables for Testing

`DropAllTables` drops all base tables in the current database. Foreign key checks are disabled during the operation so tables can be dropped in any order.

```go
err = mysqlconn.DropAllTables(ctx, conn)
```

### Example: Test Setup

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    config := &sqldb.ConnConfig{
        Driver:   mysqlconn.Driver,
        Host:     "localhost",
        Port:     3306,
        User:     "root",
        Password: "secret",
        Database: "myapp_test",
    }
    conn, err := mysqlconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
    if err != nil {
        log.Fatal(err)
    }
    db.SetConn(conn)

    // Drop all tables and recreate schema
    err = mysqlconn.DropAllTables(ctx, conn)
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
