# go-sqldb

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-sqldb.svg)](https://pkg.go.dev/github.com/domonda/go-sqldb) [![Go Report Card](https://goreportcard.com/badge/github.com/domonda/go-sqldb)](https://goreportcard.com/report/github.com/domonda/go-sqldb) [![Go](https://github.com/domonda/go-sqldb/actions/workflows/go.yml/badge.svg)](https://github.com/domonda/go-sqldb/actions/workflows/go.yml) [![Go version](https://img.shields.io/github/go-mod/go-version/domonda/go-sqldb)](https://github.com/domonda/go-sqldb) [![license](https://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://github.com/domonda/go-sqldb/blob/master/LICENSE)

## Philosophy

* Use reflection to map db rows to structs, but not as full blown ORM that replaces SQL queries (just as much ORM to increase productivity but not alienate developers who like the full power of SQL)
* Transactions are run in callback functions that can be nested
* Driver-agnostic: write code against a common `Connection` interface, swap database drivers without changing business logic
* Store the db connection and transactions in `context.Context` to pass them down into nested functions transparently

## Database drivers

| Database   | Package                                                                          | Underlying driver                                                                              |
| ---------- | -------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| PostgreSQL | [pqconn](https://pkg.go.dev/github.com/domonda/go-sqldb/pqconn)                 | [github.com/lib/pq](https://github.com/lib/pq)                                                |
| MySQL      | [mysqlconn](https://pkg.go.dev/github.com/domonda/go-sqldb/mysqlconn)           | [github.com/go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)                      |
| SQL Server | [mssqlconn](https://pkg.go.dev/github.com/domonda/go-sqldb/mssqlconn)           | [github.com/microsoft/go-mssqldb](https://github.com/microsoft/go-mssqldb)                    |
| SQLite     | [sqliteconn](https://pkg.go.dev/github.com/domonda/go-sqldb/sqliteconn)         | [zombiezen.com/go/sqlite](https://pkg.go.dev/zombiezen.com/go/sqlite)                          |
| Oracle     | [oraconn](https://pkg.go.dev/github.com/domonda/go-sqldb/oraconn)               | [github.com/sijms/go-ora/v2](https://github.com/sijms/go-ora)                                 |


### Feature matrix

| Feature                       | pqconn              | mysqlconn           | mssqlconn           | sqliteconn          | oraconn             |
| ----------------------------- | ------------------- | ------------------- | ------------------- | ------------------- | ------------------- |
| Underlying driver             | lib/pq              | go-sql-driver/mysql | go-mssqldb          | zombiezen.com/sqlite| go-ora/v2           |
| Placeholder style             | `$1`, `$2`, …       | `?`, `?`, …         | `@p1`, `@p2`, …     | `?`, `?`, …         | `:1`, `:2`, …       |
| Max query arguments           | 65 535              | 65 535              | 2 100               | 32 766              | 65 535              |
| Identifier quoting            | `"double quotes"`   | `` `backticks` ``   | `[brackets]`        | `"double quotes"`   | `"double quotes"`   |
| Default isolation level       | Read Committed      | Repeatable Read     | Read Committed      | Serializable        | Read Committed      |
| `Connection`                  | yes                 | yes                 | yes                 | yes                 | yes                 |
| `ListenerConnection`          | yes                 | —                   | —                   | —                   | —                   |
| Transactions                  | yes                 | yes                 | yes                 | yes                 | yes                 |
| Nested `Begin` uses savepoint | —                   | —                   | —                   | yes                 | —                   |
| `db.TransactionSavepoint`     | yes                 | yes                 | yes                 | yes                 | yes                 |
| Constraint error mapping      | yes                 | yes                 | yes                 | yes                 | yes                 |
| Array column support          | yes                 | —                   | —                   | —                   | —                   |
| JSON column type              | `json`, `jsonb`     | `json`              | —                   | `json`, `jsonb`     | `json`              |
| Prepared statements           | yes                 | yes                 | yes                 | yes                 | yes                 |
| `ExecRowsAffected`            | yes                 | yes                 | yes                 | yes                 | yes                 |
| `QueryBuilder`                | yes                 | yes                 | yes                 | yes                 | yes                 |
| `UpsertQueryBuilder`          | yes                 | yes                 | yes                 | yes                 | yes                 |
| `ReturningQueryBuilder`       | yes                 | —                   | —                   | yes                 | —                   |

**Notes:**
- **Nested `Begin` uses savepoint**: Only `sqliteconn` converts nested `Begin` calls into SQL `SAVEPOINT` / `RELEASE` commands. All other real drivers start a new independent transaction on the underlying connection.
- **`db.TransactionSavepoint`**: Works with any driver by issuing raw `SAVEPOINT` SQL within an existing transaction (see [Transactions](#transactions)).
- **MockConn**: In-memory mock for unit testing without a running database. Supports configurable query results, exec callbacks, and records all queries and execs for inspection.
- **ErrConn**: Dummy connection where every method except `Close` returns a stored error. Useful for testing error-handling paths.


## Query builders

Query generation is split into three interfaces to separate standard SQL from driver-specific syntax:

### `QueryBuilder` — standard SQL

Implemented by all drivers via `StdQueryBuilder`. Generates portable SQL for:
- `SELECT * FROM ... WHERE pk = $1` (QueryRowWithPK)
- `INSERT INTO ... VALUES(...)` (Insert, InsertRows)
- `UPDATE ... SET ... WHERE ...` (Update, UpdateColumns)
- `DELETE FROM ... WHERE ...` (Delete)

### `UpsertQueryBuilder` — driver-specific upsert

Not all databases use the same upsert syntax. Each driver provides its own implementation:

| Driver     | Implementation             | Syntax                                                |
| ---------- | -------------------------- | ----------------------------------------------------- |
| PostgreSQL | `pqconn.QueryBuilder`      | `INSERT ... ON CONFLICT(...) DO UPDATE SET` / `DO NOTHING` |
| SQLite     | `sqliteconn.QueryBuilder`  | Same as PostgreSQL                                    |
| MySQL      | `mysqlconn.QueryBuilder`   | `INSERT ... ON DUPLICATE KEY UPDATE col=VALUES(col)` / `col = col` (no-op for InsertUnique) |
| MSSQL      | `mssqlconn.QueryBuilder`   | `MERGE INTO ... USING ... WHEN MATCHED THEN UPDATE ... WHEN NOT MATCHED THEN INSERT ...;` |
| Oracle     | `oraconn.QueryBuilder`     | `MERGE INTO ... USING (SELECT ... FROM DUAL) ... WHEN MATCHED THEN UPDATE ... WHEN NOT MATCHED THEN INSERT ...` |

`InsertUnique` uses `ExecRowsAffected` to determine whether a row was inserted (1) or a conflict occurred (0). All drivers support this.

### `ReturningQueryBuilder` — RETURNING clause

Only PostgreSQL and SQLite support the `RETURNING` clause. `StdReturningQueryBuilder` extends `StdQueryBuilder` with:
- `InsertReturning` — `INSERT ... RETURNING ...`
- `UpdateReturning` — `UPDATE ... SET ... WHERE ... RETURNING ...`

MySQL and MSSQL query builders do not implement this interface. Functions accepting `ReturningQueryBuilder` will not compile if passed a builder that lacks support.

### Configuring the query builder

The `db` package resolves the query builder in this order:
1. Context-level override via `db.ContextWithQueryBuilder`
2. The connection from `db.Conn(ctx)` if it implements `QueryBuilder`
3. The global default set with `db.SetQueryBuilder`

All driver connections implement `QueryBuilder` automatically, so `db.SetQueryBuilder` is typically not needed:

| Connection   | Mechanism                           | `UpsertQueryBuilder` | `ReturningQueryBuilder` |
| ------------ | ----------------------------------- | :---: | :---: |
| `pqconn`     | embeds `pqconn.QueryBuilder`        | yes | yes |
| `sqliteconn` | embeds `sqliteconn.QueryBuilder`    | yes | yes |
| `mysqlconn`  | embeds `mysqlconn.QueryBuilder`     | yes | no |
| `mssqlconn`  | embeds `mssqlconn.QueryBuilder`     | yes | no |
| `oraconn`    | embeds `oraconn.QueryBuilder`       | yes | no |

PostgreSQL and SQLite connections embed their driver-specific `QueryBuilder`, which extends `StdReturningQueryBuilder` with `ON CONFLICT` upsert syntax, so the connection itself satisfies all three interfaces. MySQL, MSSQL, and Oracle embed their driver-specific builder directly, providing `QueryBuilder` and `UpsertQueryBuilder` support — but not `ReturningQueryBuilder` (Oracle's `RETURNING ... INTO` syntax is incompatible with the row-returning interface).

Driver-specific builders embed `StdQueryBuilder` and override only the methods that differ, so standard SQL operations work identically across all drivers.


## Generic errors

Each driver maps its database-specific errors to typed values defined in the root `sqldb` package:

| Type                              | Field        | Description                                   |
| --------------------------------- | ------------ | --------------------------------------------- |
| `ErrIntegrityConstraintViolation` | `Constraint` | Base type for all constraint violations       |
| `ErrNotNullViolation`             | `Constraint` | NULL inserted into a NOT NULL column          |
| `ErrUniqueViolation`              | `Constraint` | Duplicate value for a unique key              |
| `ErrForeignKeyViolation`          | `Constraint` | Referential integrity violation               |
| `ErrCheckViolation`               | `Constraint` | CHECK constraint violated                     |
| `ErrRestrictViolation`            | `Constraint` | RESTRICT constraint violated (PostgreSQL)     |
| `ErrExclusionViolation`           | `Constraint` | Exclusion constraint violated (PostgreSQL)    |
| `ErrDeadlock`                     | —            | Deadlock detected between transactions        |
| `ErrSerializationFailure`         | —            | Transaction serialization conflict (retry)    |
| `ErrRaisedException`              | `Message`    | User-defined exception (RAISE/SIGNAL/THROW)   |

All specific types unwrap to `ErrIntegrityConstraintViolation`, so `errors.As` traverses the chain and matches any subtype:

```go
// catch any constraint violation and read the constraint name
var cv sqldb.ErrIntegrityConstraintViolation
if errors.As(err, &cv) {
    fmt.Println("constraint violated:", cv.Constraint)
}

// catch a specific violation type
var uv sqldb.ErrUniqueViolation
if errors.As(err, &uv) {
    fmt.Println("unique constraint violated:", uv.Constraint)
}
```

### Error mapping matrix

| Error type                        | pqconn | mysqlconn | mssqlconn | sqliteconn | oraconn |
| --------------------------------- | ------ | --------- | --------- | ---------- | ------- |
| `ErrIntegrityConstraintViolation` | yes    | —         | —         | yes        | —       |
| `ErrNotNullViolation`             | yes    | yes       | yes       | yes        | yes     |
| `ErrUniqueViolation`              | yes    | yes       | yes       | yes        | yes     |
| `ErrForeignKeyViolation`          | yes    | yes       | yes       | yes        | yes     |
| `ErrCheckViolation`               | yes    | yes       | yes       | yes        | yes     |
| `ErrRestrictViolation`            | yes    | —         | —         | —          | —       |
| `ErrExclusionViolation`           | yes    | —         | —         | —          | —       |
| `ErrDeadlock`                     | yes    | yes       | yes       | —          | yes     |
| `ErrSerializationFailure`         | yes    | —         | —         | —          | yes     |
| `ErrRaisedException`              | yes    | yes       | yes       | —          | yes     |

Driver packages also expose driver-specific helper functions (e.g. `pqconn.IsUniqueViolation`) for error conditions that have no generic `sqldb` type, such as query cancellations or text-representation errors. See each driver's README for the full list.


## Usage

The recommended way to use this library is through the [github.com/domonda/go-sqldb/db](https://pkg.go.dev/github.com/domonda/go-sqldb/db)
package. Every function just takes a `ctx` and the `db` package retrieves the right connection automatically:
first from the context (e.g. a transaction injected by `db.Transaction`), then falling back to the global connection set with `db.SetConn`.

See the [db package README](db/README.md) for a complete function reference and usage patterns.

### Creating a connection

```go
config := &sqldb.ConnConfig{
    Driver:   "postgres",
    Host:     "localhost",
    User:     "postgres",
    Database: "demo",
    Extra:    map[string]string{"sslmode": "disable"},
}

conn, err := pqconn.Connect(ctx, config)
if err != nil {
    panic(err)
}
defer conn.Close()

// Set as the global connection used by the db package
db.SetConn(conn)
```

### Struct field mapping

The default `StructReflector` maps struct fields to database columns using the `db` struct tag:

```go
type User struct {
    ID    uu.ID  `db:"id,primarykey"`
    Email string `db:"email"`
    Name  string `db:"name"`
    // Field with tag "-" will be ignored
    Internal string `db:"-"`
}
```

Available tag options:

| Tag                            | Meaning                                             |
| ------------------------------ | --------------------------------------------------- |
| `db:"column_name"`             | Map field to column                                 |
| `db:"column_name,primarykey"`  | Mark as primary key (required for update and upsert)|
| `db:"column_name,readonly"`    | Excluded from INSERT and UPDATE                     |
| `db:"column_name,default"`     | Has a database default, can be ignored on INSERT    |
| `db:"-"`                       | Ignore field entirely                               |

For struct-based insert, update, and upsert operations the struct must embed `sqldb.TableName`
with a `db` tag to specify the target table:

```go
type User struct {
    sqldb.TableName `db:"public.user"`

    ID        uu.ID  `db:"id,primarykey,default"`
    Email     string `db:"email"`
    Name      string `db:"name"`
    CreatedAt time.Time `db:"created_at,readonly,default"`
}
```

You can customize the struct reflector globally or per context:

```go
reflector := &sqldb.TaggedStructReflector{
    NameTag:          "col",           // Use "col" tag instead of "db"
    Ignore:           "_ignore_",      // Ignore fields with this value
    PrimaryKey:       "pk",
    ReadOnly:         "readonly",
    Default:          "default",
    UntaggedNameFunc: sqldb.ToSnakeCase, // Convert untagged fields to snake_case
}

// Set globally
db.SetStructReflector(reflector)

// Or set per context
ctx = db.ContextWithStructReflector(ctx, reflector)
```

### Slice and array column handling

Slice and array column handling (like PostgreSQL arrays) is handled transparently by driver implementations. For example, the `pqconn` driver automatically wraps Go slices and arrays with `pq.Array()` for both query arguments and row scanning, so you can use native Go slices in structs mapped to PostgreSQL array columns without any manual conversion.

### Exec

```go
err = db.Exec(ctx, /*sql*/ `DELETE FROM public.user WHERE id = $1`, userID)
```

### ExecRowsAffected

```go
n, err := db.ExecRowsAffected(ctx, /*sql*/ `UPDATE public.user SET name = $1 WHERE active = $2`, "Inactive", false)
fmt.Printf("%d rows updated\n", n)
```

### Querying a single row

```go
// Scan into a struct
user, err := db.QueryRowAs[User](ctx,
    /*sql*/ `SELECT * FROM public.user WHERE id = $1`,
    userID,
)

// Scan a scalar value
var count int64
count, err = db.QueryRowAs[int64](ctx, /*sql*/ `SELECT count(*) FROM public.user`)

// Return a default value instead of sql.ErrNoRows
user, err = db.QueryRowAsOr(ctx, defaultUser,
    /*sql*/ `SELECT * FROM public.user WHERE id = $1`,
    userID,
)

// Low-level: scan into individual variables
var name string
var email string
err = db.QueryRow(ctx,
    /*sql*/ `SELECT name, email FROM public.user WHERE id = $1`,
    userID,
).Scan(&name, &email)
```

### Querying a single row by primary key

For structs with an embedded `sqldb.TableName`, you can query by primary key directly:

```go
user, err := db.QueryRowByPK[User](ctx, userID)

// Return a default value instead of sql.ErrNoRows
user, err = db.QueryRowByPKOr(ctx, defaultUser, userID)
```

### Querying multiple rows

```go
// Query into a slice of structs
users, err := db.QueryRowsAsSlice[User](ctx, /*sql*/ `SELECT * FROM public.user`)

// Query a single column into a scalar slice
emails, err := db.QueryRowsAsSlice[string](ctx, /*sql*/ `SELECT email FROM public.user`)
```

### QueryCallback for per-row processing

```go
// Callback arguments are scanned from columns via reflection
err = db.QueryCallback(ctx,
    func(name, email string) {
        fmt.Printf("%q <%s>\n", name, email)
    },
    /*sql*/ `SELECT name, email FROM public.user`,
)

// With context and error return
err = db.QueryCallback(ctx,
    func(ctx context.Context, user *User) error {
        return processUser(ctx, user)
    },
    /*sql*/ `SELECT * FROM public.user`,
)

// Typed struct callback (generic, no reflection on the callback signature)
err = db.QueryStructCallback[User](ctx,
    func(user User) error {
        return processUser(ctx, user)
    },
    /*sql*/ `SELECT * FROM public.user`,
)
```

### Insert

```go
// Insert a struct (table name from embedded sqldb.TableName)
newUser := &User{Name: "Alice", Email: "alice@example.com"}
err = db.InsertRowStruct(ctx, newUser)

// Ignore columns with database defaults
err = db.InsertRowStruct(ctx, newUser, sqldb.IgnoreColumns("id", "created_at"))

// Insert using a values map
err = db.Insert(ctx, "public.user", sqldb.Values{
    "name":  "Erik Unger",
    "email": "erik@domonda.com",
})

// Insert with RETURNING clause
var id uu.ID
err = db.InsertReturning(ctx, "public.user", sqldb.Values{
    "name":  "Erik Unger",
    "email": "erik@domonda.com",
}, "id").Scan(&id)

// Insert or do nothing on conflict
inserted, err := db.InsertUnique(ctx, "public.user", sqldb.Values{
    "email": "erik@domonda.com",
    "name":  "Erik Unger",
}, "ON CONFLICT (email) DO NOTHING")

// Batch insert a slice of structs (uses a transaction + prepared statement)
err = db.InsertRowStructs(ctx, users)
```

### Update

```go
// Update with a values map and WHERE clause
err = db.Update(ctx, "public.user", sqldb.Values{"name": "New Name"},
    /*sql*/ `WHERE id = $1`, userID,
)

// Update using a struct (WHERE clause built from primarykey fields)
err = db.UpdateRowStruct(ctx, &user)

// Update only specific columns
err = db.UpdateRowStruct(ctx, &user, sqldb.OnlyColumns("name", "email"))
```

### Upsert

Insert or update on primary key conflict:

```go
// Upsert a struct
err = db.UpsertRowStruct(ctx, &user)

// Upsert ignoring certain columns
err = db.UpsertRowStruct(ctx, &user, sqldb.IgnoreColumns("created_at"))

// Batch upsert a slice of structs
err = db.UpsertRowStructs(ctx, users)
```

### Transactions

Functions called within a transaction automatically use the transaction connection
via `db.Conn(ctx)`, without needing to know whether they are inside a transaction or not.

```go
func GetUserOrNil(ctx context.Context, userID uu.ID) (user *User, err error) {
    err = db.QueryRow(ctx,
        /*sql*/ `SELECT * FROM public.user WHERE id = $1`,
        userID,
    ).Scan(&user)
    if err != nil {
        return nil, db.ReplaceErrNoRows(err, nil)
    }
    return user, nil
}

func CreateOrUpdateUser(ctx context.Context, userID uu.ID) error {
    // GetUserOrNil transparently uses the transaction connection
    return db.Transaction(ctx, func(ctx context.Context) error {
        user, err := GetUserOrNil(ctx, userID)
        if err != nil {
            return err
        }
        if user == nil {
            return db.InsertRowStruct(ctx, &User{ID: userID, Name: "New"})
        }
        return db.Exec(ctx, /*sql*/ `UPDATE public.user SET name = $1 WHERE id = $2`, "Updated", userID)
    })
}
```

Transaction variants:

```go
// With explicit options
err = db.TransactionOpts(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(ctx context.Context) error { ... })

// Read-only
err = db.TransactionReadOnly(ctx, func(ctx context.Context) error { ... })

// Return a value from a transaction
user, err := db.TransactionResult[User](ctx, func(ctx context.Context) (User, error) { ... })

// Serialized with automatic retry on serialization failure
err = db.SerializedTransaction(ctx, func(ctx context.Context) error { ... })

// Savepoints for nested partial rollback
err = db.TransactionSavepoint(ctx, func(ctx context.Context) error { ... })

// Skip the transaction (useful for debugging)
err = db.DebugNoTransaction(ctx, func(ctx context.Context) error { ... })
```

### Prepared statements

```go
// Prepared query statement
queryUser, closeStmt, err := db.QueryRowAsStmt[User](ctx, /*sql*/ `SELECT * FROM public.user WHERE id = $1`)
if err != nil {
    return err
}
defer closeStmt()

user, err := queryUser(ctx, userID)
```

### LISTEN/NOTIFY (PostgreSQL)

```go
err = db.ListenOnChannel(ctx, "user_changes",
    func(channel, payload string) {
        fmt.Printf("Notification on %s: %s\n", channel, payload)
    },
    func(channel string) {
        fmt.Printf("Unlistened from %s\n", channel)
    },
)

// Later...
err = db.UnlistenChannel(ctx, "user_changes")
```

Returns `errors.ErrUnsupported` if the connection does not implement `ListenerConnection`.

### Query options

Filter which struct fields are included in insert, update, and upsert operations:

```go
// Ignore specific columns
db.InsertRowStruct(ctx, &user, sqldb.IgnoreColumns("id", "created_at"))

// Include only specific columns
db.UpdateRowStruct(ctx, &user, sqldb.OnlyColumns("name", "email"))

// Ignore by struct field name
db.InsertRowStruct(ctx, &user, sqldb.IgnoreStructFields("Internal"))

// Built-in filters
sqldb.IgnoreHasDefault  // Ignore columns with the "default" tag option
sqldb.IgnorePrimaryKey  // Ignore primary key columns
sqldb.IgnoreReadOnly    // Ignore read-only columns (applied automatically for insert/update)
```


## Low-level API

The root `sqldb` package exposes the same operations as the `db` package but with explicit connection, reflector, builder, and formatter arguments. This is useful when you need full control or are building your own abstractions:

```go
user, err := sqldb.QueryRowAs[User](ctx, conn, reflector, conn, /*sql*/ `SELECT * FROM public.user WHERE id = $1`, userID)

err = sqldb.InsertRowStruct(ctx, conn, reflector, queryBuilder, conn, &user)

err = sqldb.Transaction(ctx, conn, &sql.TxOptions{ReadOnly: true}, func(tx sqldb.Connection) error {
    return tx.Exec(ctx, /*sql*/ `UPDATE public.user SET name = $1 WHERE id = $2`, "Alice", userID)
})
```

Driver `Connect` functions return types that implement the `Connection` interface, which embeds `QueryFormatter`.


## Internal caching

The package internally caches struct reflection data and generated SQL queries to avoid repeated reflection and string building on every call. Caches are keyed by struct type, `StructReflector`, `QueryBuilder`, and `QueryFormatter` and are protected by `sync.RWMutex` for concurrent use.

Cached data includes:
- **Struct reflection**: Flattened field metadata (column names, flags, field indices) for each struct type and reflector combination.
- **INSERT queries**: The generated SQL query string and struct field indices, cached per struct type and connection configuration.
- **UPSERT queries**: Same as INSERT caching for upsert operations.
- **QueryRowByPK queries**: The generated SELECT query and primary key column count.

Query caches are bypassed when `QueryOption` arguments are provided, since options like `ColumnFilter` change which columns are included and are not part of the cache key.

All caches can be cleared with `ClearQueryCaches()` which is useful for testing and debugging.


## Testing

### MockConn for unit tests

`MockConn` implements `ListenerConnection` and `QueryFormatter` entirely in memory, allowing you to unit test database-dependent code without a running database.

#### Creating a MockConn

```go
// Create a MockConn for PostgreSQL-style $1, $2, ... placeholders.
mockConn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
```

Use builder methods to configure further:
- `WithNormalizeQuery`: set a `NormalizeQueryFunc` to normalize SQL whitespace before matching
- `WithQueryLog`: set an `io.Writer` to log all executed SQL statements

#### Registering mock query results

Use `WithQueryResult` to register expected results for specific queries. It returns a cloned `MockConn` so you can chain calls:

```go
mockConn = mockConn.WithQueryResult(
    []string{"id", "email", "name"},               // column names
    [][]driver.Value{                               // rows
        {"550e8400-e29b-41d4-a716-446655440000", "alice@example.com", "Alice"},
        {"6ba7b810-9dad-11d1-80b4-00c04fd430c8", "bob@example.com", "Bob"},
    },
    `SELECT id, email, name FROM public.user`,      // the query to match
    // args... (if the query has placeholders)
)
```

For queries with arguments:

```go
mockConn = mockConn.WithQueryResult(
    []string{"id", "email", "name"},
    [][]driver.Value{
        {"550e8400-e29b-41d4-a716-446655440000", "alice@example.com", "Alice"},
    },
    `SELECT id, email, name FROM public.user WHERE id = $1`,
    "550e8400-e29b-41d4-a716-446655440000",         // matches $1
)
```

If a query has no matching result registered, the returned `Rows` will have an error wrapping `sql.ErrNoRows`.

#### Using MockConn with the db package

`MockConn` implements `Connection`, so it can be used directly with the `db` package:

```go
func TestGetUser(t *testing.T) {
    mockConn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
        WithQueryResult(
            []string{"id", "email", "name"},
            [][]driver.Value{
                {"550e8400-e29b-41d4-a716-446655440000", "alice@example.com", "Alice"},
            },
            `SELECT id, email, name FROM public.user WHERE id = $1`,
            "550e8400-e29b-41d4-a716-446655440000",
        )

    ctx := db.ContextWithConn(t.Context(), mockConn)

    user, err := GetUser(ctx, uu.IDFrom("550e8400-e29b-41d4-a716-446655440000"))
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
    assert.Equal(t, "alice@example.com", user.Email)
}
```

#### Mocking Exec calls

By default, `Exec` returns the context error (nil for non-canceled contexts). To customize:

```go
mockConn.MockExec = func(ctx context.Context, query string, args ...any) error {
    if strings.Contains(query, "DELETE") {
        return errs.New("delete not allowed in test")
    }
    return nil
}
```

#### Custom query handling with MockQuery

For dynamic query responses, set the `MockQuery` function instead of using `WithQueryResult`:

```go
mockConn.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
    if strings.Contains(query, "public.user") {
        return sqldb.NewMockRows("id", "name").
            WithRow("some-id", "Alice")
    }
    return sqldb.NewErrRows(sql.ErrNoRows)
}
```

Note: when `MockQuery` is set, `WithQueryResult` results are not consulted.

#### Mocking transactions

Transactions work out of the box. `Begin` returns a copy of the `MockConn` with the transaction ID set, and `Commit`/`Rollback` return nil:

```go
func TestWithTransaction(t *testing.T) {
    mockConn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
        WithQueryResult(
            []string{"count"},
            [][]driver.Value{{int64(42)}},
            `SELECT count(*) FROM public.user`,
        )

    ctx := db.ContextWithConn(t.Context(), mockConn)

    err := db.Transaction(ctx, func(ctx context.Context) error {
        count, err := db.QueryRowAs[int64](ctx, `SELECT count(*) FROM public.user`)
        require.NoError(t, err)
        assert.Equal(t, int64(42), count)
        return nil
    })
    require.NoError(t, err)
}
```

#### Inspecting recorded queries

All queries and exec calls are recorded in the `Recordings` field:

```go
// After running code under test...
require.Len(t, mockConn.Recordings.Queries, 1)
assert.Equal(t, `SELECT id FROM public.user WHERE email = $1`, mockConn.Recordings.Queries[0].Query)

require.Len(t, mockConn.Recordings.Execs, 1)
assert.Contains(t, mockConn.Recordings.Execs[0].Query, "UPDATE")
```

#### Logging SQL for debugging

Pass an `io.Writer` to log all SQL statements:

```go
var buf strings.Builder
mockConn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
    WithQueryLog(&buf)

// ... run code under test ...

t.Log("Executed SQL:\n" + buf.String())
```

### Integration tests

Integration tests use dockerized database instances to avoid conflicts with local installations:

| Driver    | Database        | Port | Docker Compose                         |
| --------- | --------------- | ---- | -------------------------------------- |
| pqconn    | PostgreSQL 17   | 5433 | `pqconn/test/docker-compose.yml`       |
| mysqlconn | MariaDB 11.7    | 3307 | `mysqlconn/test/docker-compose.yml`    |
| mssqlconn | SQL Server 2022 | 1434 | `mssqlconn/test/docker-compose.yml`    |
| oraconn   | Oracle Free 23  | 1522 | `oraconn/test/docker-compose.yml`      |

Start a test database and run all tests:
```bash
docker compose -f pqconn/test/docker-compose.yml up -d
./test-workspace.sh
```

After changing a database version in `docker-compose.yml`, reset the data directory:
```bash
./pqconn/test/reset-postgres-data.sh
./mysqlconn/test/reset-mariadb-data.sh
./mssqlconn/test/reset-mssql-data.sh
# oraconn has no persistent data — use: docker compose -f oraconn/test/docker-compose.yml down
```

## History

This package started out as an extension wrapper of [github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx) but turned into a complete rewrite using the same philosophy of representing table rows as Go structs.

It has been used and refined for years in production by [domonda](https://domonda.com) using the database driver [github.com/lib/pq](https://github.com/lib/pq).

The design patterns evolved mostly through discovery led by the desire to minimize boilerplate code while maintaining the full power of SQL.

## License

[MIT](LICENSE)

