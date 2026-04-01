# go-sqldb

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-sqldb.svg)](https://pkg.go.dev/github.com/domonda/go-sqldb) [![Go Report Card](https://goreportcard.com/badge/github.com/domonda/go-sqldb)](https://goreportcard.com/report/github.com/domonda/go-sqldb) [![Go](https://github.com/domonda/go-sqldb/actions/workflows/go.yml/badge.svg)](https://github.com/domonda/go-sqldb/actions/workflows/go.yml) [![Go version](https://img.shields.io/github/go-mod/go-version/domonda/go-sqldb)](https://github.com/domonda/go-sqldb) [![license](https://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://github.com/domonda/go-sqldb/blob/master/LICENSE)

## Table of contents

- [Philosophy](#philosophy)
- [Database drivers](#database-drivers)
  - [Feature matrix](#feature-matrix)
- [Query builders](#query-builders)
  - [`QueryBuilder` — standard SQL](#querybuilder--standard-sql)
  - [`UpsertQueryBuilder` — driver-specific upsert](#upsertquerybuilder--driver-specific-upsert)
  - [`ReturningQueryBuilder` — RETURNING clause](#returningquerybuilder--returning-clause)
  - [Configuring the query builder](#configuring-the-query-builder)
- [Generic errors](#generic-errors)
  - [Error mapping matrix](#error-mapping-matrix)
- [Usage](#usage)
  - [Creating a connection](#creating-a-connection)
  - [Struct field mapping](#struct-field-mapping)
  - [Column and struct field mismatch behavior](#column-and-struct-field-mismatch-behavior)
  - [Type wrappers](#type-wrappers)
  - [Exec](#exec)
  - [ExecRowsAffected](#execrowsaffected)
  - [Querying a single row](#querying-a-single-row)
  - [Querying a single row by primary key](#querying-a-single-row-by-primary-key)
  - [Querying multiple rows](#querying-multiple-rows)
  - [QueryCallback for per-row processing](#querycallback-for-per-row-processing)
  - [Insert](#insert)
  - [Update](#update)
  - [Upsert](#upsert)
  - [Transactions](#transactions)
  - [Prepared statements](#prepared-statements)
  - [LISTEN/NOTIFY (PostgreSQL)](#listennotify-postgresql)
  - [Query options](#query-options)
- [Low-level API](#low-level-api)
- [Internal caching](#internal-caching)
- [Performance optimizations](#performance-optimizations)
  - [Struct reflection caching](#struct-reflection-caching)
  - [Batch insert optimization (`InsertRowStructs`)](#batch-insert-optimization-insertrowstructs)
  - [Batch update and delete optimization (`UpdateRowStructs`, `DeleteRowStructs`)](#batch-update-and-delete-optimization-updaterowstructs-deleterowstructs)
  - [Transaction nesting avoidance](#transaction-nesting-avoidance)
- [Testing](#testing)
  - [MockConn for unit tests](#mockconn-for-unit-tests)
  - [Integration tests](#integration-tests)
    - [Shared test suite (`conntest`)](#shared-test-suite-conntest)
- [History](#history)
- [License](#license)

## Philosophy

* Use reflection to map db rows to structs, but not as full blown ORM that replaces SQL queries (just as much ORM to increase productivity but not alienate developers who like the full power of SQL)
* Transactions are run in callback functions that can be nested
* Driver-agnostic: write code against a common `Connection` interface, swap database drivers without changing business logic
* Store the db connection and transactions in `context.Context` to pass them down into nested functions transparently

## Database drivers

| Database   | Package                                                                          | Underlying driver                                                                              |
| ---------- | -------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| PostgreSQL    | [pqconn](https://pkg.go.dev/github.com/domonda/go-sqldb/pqconn)                 | [github.com/lib/pq](https://github.com/lib/pq)                                                |
| MySQL/MariaDB | [mysqlconn](https://pkg.go.dev/github.com/domonda/go-sqldb/mysqlconn)           | [github.com/go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)                      |
| MS SQL Server | [mssqlconn](https://pkg.go.dev/github.com/domonda/go-sqldb/mssqlconn)           | [github.com/microsoft/go-mssqldb](https://github.com/microsoft/go-mssqldb)                    |
| SQLite        | [sqliteconn](https://pkg.go.dev/github.com/domonda/go-sqldb/sqliteconn)         | [zombiezen.com/go/sqlite](https://pkg.go.dev/zombiezen.com/go/sqlite)                          |
| Oracle        | [oraconn](https://pkg.go.dev/github.com/domonda/go-sqldb/oraconn)               | [github.com/sijms/go-ora/v2](https://github.com/sijms/go-ora)                                 |


### Feature matrix

| Feature                       | pqconn              | mysqlconn           | mssqlconn           | sqliteconn          | oraconn             |
| ----------------------------- | ------------------- | ------------------- | ------------------- | ------------------- | ------------------- |
| Underlying driver             | lib/pq              | go-sql-driver/mysql | go-mssqldb          | zombiezen.com/sqlite| go-ora/v2           |
| Placeholder style             | `$1`, `$2`, …       | `?`, `?`, …         | `@p1`, `@p2`, …     | `?1`, `?2`, …       | `:1`, `:2`, …       |
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

`UpdateColumns` numbers placeholders sequentially: SET columns first, then WHERE (primary key) columns. MySQL and Oracle override `Update` to reorder arguments for their positional placeholder binding.

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
config := &sqldb.Config{
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

For struct-based insert, update, and upsert operations the struct must embed `db.TableName`
with a `db` tag to specify the target table:

```go
type User struct {
    db.TableName `db:"public.user"`

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

### Column and struct field mismatch behavior

When scanning query results into structs, the number of query result columns and mapped struct fields do not need to match exactly:

**Query returns fewer columns than the struct has mapped fields:**
Struct fields with no corresponding result column are silently skipped and left unchanged. This means you can use `SELECT col1, col2 FROM ...` with a struct that maps ten columns — only the two selected columns will be scanned into, while the remaining fields retain whatever value they had before scanning.

**Query returns columns not mapped to any struct field:**
By default, unmapped result columns are silently discarded during scanning. This is convenient when using `SELECT *` with structs that don't cover every column.

To catch this as an error instead, set `FailOnUnmappedColumns` to `true` on the `TaggedStructReflector`:

```go
reflector := sqldb.NewTaggedStructReflector()
reflector.FailOnUnmappedColumns = true
db.SetStructReflector(reflector)
```

With `FailOnUnmappedColumns` enabled, scanning will return an error listing all result columns that have no corresponding struct field. This is useful for catching schema drift or accidental `SELECT *` queries that return unexpected columns.

Similarly, `FailOnUnmappedStructFields` catches the reverse: struct fields that have no corresponding column in the query result. This is useful for catching incomplete `SELECT` queries that accidentally omit columns:

```go
reflector := sqldb.NewTaggedStructReflector()
reflector.FailOnUnmappedStructFields = true
db.SetStructReflector(reflector)
```

Both flags can be enabled together for strict bidirectional checking where every query result column must map to a struct field and every struct field must have a corresponding query result column.

### Type wrappers

Type wrappers let you customize how Go types are serialized to and deserialized from database columns. A `TypeWrapper` implements two methods: `WrapAsScanner` for reading (returns an `sql.Scanner`) and `WrapAsValuer` for writing (returns a `driver.Valuer`). Each method returns nil if the wrapper does not handle the given type, allowing multiple wrappers to be composed.

Pass type wrappers to `NewTaggedStructReflector`:

```go
reflector := sqldb.NewTaggedStructReflector(
    sqldb.MailAddressTypeWrapper{},
    myCustomTypeWrapper{},
)
db.SetStructReflector(reflector)
```

Or set them on an existing reflector:

```go
reflector := sqldb.NewTaggedStructReflector()
reflector.TypeWrappers = sqldb.TypeWrappers{sqldb.MailAddressTypeWrapper{}}
```

#### Built-in type wrappers

| Type wrapper             | Handles                        | Scanner behavior                                          | Valuer behavior                  |
| ------------------------ | ------------------------------ | --------------------------------------------------------- | -------------------------------- |
| `MailAddressTypeWrapper`  | `mail.Address`, `*mail.Address` | Parses RFC 5322 address via `mail.ParseAddress`; NULL → zero/nil | `mail.Address.String()`; nil → NULL |

#### Driver-level wrapping

Some type conversions are handled at the driver level rather than through `TypeWrapper`. For example, `pqconn` automatically wraps Go slices and arrays with `pq.Array()` for both query arguments and scan destinations, so PostgreSQL array columns work transparently without a type wrapper. This applies to all slices and arrays except `[]byte` (treated as a string) and types that already implement `driver.Valuer` or `sql.Scanner`.

#### Implementing a custom type wrapper

```go
type moneyTypeWrapper struct{}

func (moneyTypeWrapper) WrapAsScanner(val reflect.Value) sql.Scanner {
    if val.Type() != reflect.TypeFor[Money]() {
        return nil // not handled
    }
    return &moneyScanner{ptr: val.Addr()}
}

func (moneyTypeWrapper) WrapAsValuer(val reflect.Value) driver.Valuer {
    if val.Type() != reflect.TypeFor[Money]() {
        return nil // not handled
    }
    return moneyValuer{val: val.Interface().(Money)}
}
```

### Exec

```go
err = db.Exec(ctx, `DELETE FROM public.user WHERE id = $1`, userID)
```

### ExecRowsAffected

```go
n, err := db.ExecRowsAffected(ctx, 
    `UPDATE public.user SET name = $1 WHERE active = $2`,
    "Inactive", false,
)
fmt.Printf("%d rows updated\n", n)
```

### Querying a single row

```go
// Scan into a struct
user, err := db.QueryRowAs[User](ctx,
    `SELECT * FROM public.user WHERE id = $1`, userID,
)

// Scan a scalar value
count, err := db.QueryRowAs[int](ctx, `SELECT count(*) FROM public.user`)

// Return a default value instead of sql.ErrNoRows
user, err = db.QueryRowAsOr(ctx, defaultUser,
    `SELECT * FROM public.user WHERE id = $1`, userID,
)

// Scan multiple scalar values with generics
name, email, err := db.QueryRowAs2[string, *mail.Address](ctx,
    `SELECT name, email FROM public.user WHERE id = $1`, userID,
)

// Low-level: scan into individual variables
var (
    name  string
    email *mail.Address
)
err = db.QueryRow(ctx,
    `SELECT name, email FROM public.user WHERE id = $1`, userID,
).Scan(&name, &email)
```

### Querying a single row by primary key

For structs with an embedded `db.TableName`, you can query by primary key directly:

```go
user, err := db.QueryRowStruct[User](ctx, userID)

// Return a default value instead of sql.ErrNoRows
user, err = db.QueryRowStructOr(ctx, defaultUser, userID)
```

### Querying multiple rows

```go
// Query into a slice of structs
users, err := db.QueryRowsAsSlice[User](ctx, `SELECT * FROM public.user`)

// Query a single column into a scalar slice
emails, err := db.QueryRowsAsSlice[string](ctx, `SELECT email FROM public.user`)
```

### QueryCallback for per-row processing

```go
// Callback arguments are scanned from columns via reflection
err = db.QueryCallback(ctx,
    func(name, email string) {
        fmt.Printf("%q <%s>\n", name, email)
    },
    `SELECT name, email FROM public.user`,
)

// With context and error return
err = db.QueryCallback(ctx,
    func(ctx context.Context, user *User) error {
        return processUser(ctx, user)
    },
    `SELECT * FROM public.user`,
)

// Typed struct callback (generic, no reflection on the callback signature)
err = db.QueryStructCallback[User](ctx,
    func(user User) error {
        return processUser(ctx, user)
    },
    `SELECT * FROM public.user`,
)
```

### Insert

```go
// Insert a struct (table name from embedded db.TableName)
newUser := &User{Name: "Alice", Email: "alice@example.com"}
err = db.InsertRowStruct(ctx, newUser)

// Ignore columns with database defaults
err = db.InsertRowStruct(ctx, newUser, db.IgnoreColumns("id", "created_at"))

// Insert using a values map
err = db.Insert(ctx, "public.user", db.Values{
    "name":  "Erik Unger",
    "email": "erik@example.com",
})

// Insert with RETURNING clause
var id uu.ID
err = db.InsertReturning(ctx, "public.user", db.Values{
    "name":  "Erik Unger",
    "email": "erik@example.com",
}, "id").Scan(&id)

// Insert or do nothing on conflict
inserted, err := db.InsertUnique(ctx, "public.user", db.Values{
    "email": "erik@example.com",
    "name":  "Erik Unger",
}, `ON CONFLICT (email) DO NOTHING`)

// Batch insert a slice of structs (uses a transaction + prepared statement)
err = db.InsertRowStructs(ctx, users)
```

### Update

```go
// Update with a values map and WHERE clause
err = db.Update(ctx, "public.user", db.Values{"name": "New Name"},
    `WHERE id = $1`, userID,
)

// Update using a struct (WHERE clause built from primarykey fields)
err = db.UpdateRowStruct(ctx, &user)

// Update only specific columns
err = db.UpdateRowStruct(ctx, &user, db.OnlyColumns("name", "email"))
```

### Upsert

Insert or update on primary key conflict:

```go
// Upsert a struct
err = db.UpsertRowStruct(ctx, &user)

// Upsert ignoring certain columns
err = db.UpsertRowStruct(ctx, &user, db.IgnoreColumns("created_at"))

// Batch upsert a slice of structs
err = db.UpsertRowStructs(ctx, users)
```

### Transactions

Functions called within a transaction automatically use the transaction connection
via `db.Conn(ctx)`, without needing to know whether they are inside a transaction or not.

```go
func GetUserOrNil(ctx context.Context, userID uu.ID) (user *User, err error) {
    err = db.QueryRow(ctx,
        `SELECT * FROM public.user WHERE id = $1`, userID,
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
        return db.Exec(ctx, `UPDATE public.user SET name = $1 WHERE id = $2`, "Updated", userID)
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
queryUser, closeStmt, err := db.QueryRowAsStmt[User](ctx, `SELECT * FROM public.user WHERE id = $1`)
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
db.InsertRowStruct(ctx, &user, db.IgnoreColumns("id", "created_at"))

// Include only specific columns
db.UpdateRowStruct(ctx, &user, db.OnlyColumns("name", "email"))

// Ignore by struct field name
db.InsertRowStruct(ctx, &user, db.IgnoreStructFields("Internal"))

// Built-in filters
db.IgnoreHasDefault  // Ignore columns with the "default" tag option
db.IgnorePrimaryKey  // Ignore primary key columns
db.IgnoreReadOnly    // Ignore read-only columns (applied automatically for insert/update)
```


## Low-level API

The root `sqldb` package exposes the same operations as the `db` package but with explicit connection, reflector, builder, and formatter arguments. This is useful when you need full control or are building your own abstractions:

```go
user, err := sqldb.QueryRowAs[User](ctx, conn, reflector, conn, `SELECT * FROM public.user WHERE id = $1`, userID)

err = sqldb.InsertRowStruct(ctx, conn, reflector, queryBuilder, conn, &user)

err = sqldb.Transaction(ctx, conn, &sql.TxOptions{ReadOnly: true}, func(tx sqldb.Connection) error {
    return tx.Exec(ctx, `UPDATE public.user SET name = $1 WHERE id = $2`, "Alice", userID)
})
```

Driver `Connect` functions return types that implement the `Connection` interface, which embeds `QueryFormatter`.


## Internal caching

The package internally caches struct reflection data and generated SQL queries to avoid repeated reflection and string building on every call. Caches are keyed by struct type, `StructReflector`, `QueryBuilder`, and `QueryFormatter` and are protected by `sync.RWMutex` for concurrent use.

Cached data includes:
- **Struct reflection**: Flattened field metadata (column names, flags, field indices) for each struct type and reflector combination.
- **INSERT queries**: The generated SQL query string and struct field indices, cached per struct type and connection configuration.
- **UPDATE queries**: The generated SQL query string and struct field indices (reordered: non-PK first, then PK), cached per struct type and connection configuration.
- **UPSERT queries**: Same as INSERT caching for upsert operations.
- **QueryRowStruct queries**: The generated SELECT query and primary key column count.

Query caches are bypassed when `QueryOption` arguments are provided, since options like `ColumnFilter` change which columns are included and are not part of the cache key.

All caches can be cleared with `ClearQueryCaches()` which is useful for testing and debugging.


## Performance optimizations

### Struct reflection caching

Struct reflection is expensive, so the package caches all reflected struct metadata on first use. The reflection cache stores flattened field metadata (column names, flags, multi-level field indices) keyed by the struct's `reflect.Type` and `StructReflector`. A read-lock fast path serves cached entries without contention; a write-lock slow path builds and stores entries on cache miss. Subsequent operations on the same struct type skip reflection entirely and use the cached field indices to extract values directly via `reflect.Value.FieldByIndex`.

The same caching principle applies to generated SQL strings. Each struct-based operation (insert, update, upsert, delete, query) caches its generated query along with the struct field indices needed to collect argument values. On cache hit, the operation jumps straight to value extraction and query execution — no reflection, no string building.

### Batch insert optimization (`InsertRowStructs`)

`InsertRowStructs` uses a multi-level optimization strategy for inserting slices of structs:

1. **Single row**: Delegates to `InsertRowStruct`, which benefits from the query cache described above.
2. **Single batch** (all rows fit within `MaxArgs()`): Generates a single multi-row `INSERT INTO ... VALUES (...), (...), ...` statement and executes it directly — no transaction, no prepared statement.
3. **Multiple batches**: Wraps all batches in a transaction for atomicity. The batch size is calculated as `MaxArgs() / numColumns` to maximize rows per statement while staying within the driver's parameter limit.
   - When there are **2 or more full batches**, a prepared statement is created for the full-batch query and reused across all full batches. This avoids repeated query parsing and planning on the database server.
   - A **single full batch** is executed directly without preparing.
   - Any **remainder rows** (fewer than a full batch) are executed as a separate, smaller multi-row INSERT.

This approach minimizes both round-trips to the database and per-statement overhead, while respecting each driver's maximum argument limit (e.g. 65,535 for PostgreSQL, 2,100 for SQL Server).

### Batch update and delete optimization (`UpdateRowStructs`, `DeleteRowStructs`)

`UpdateRowStructs` and `DeleteRowStructs` follow the same pattern as `InsertRowStructs`: all operations are wrapped in a transaction for atomicity, and a prepared statement is created once and reused across all rows. For a single row, both functions delegate to their single-row counterpart (`UpdateRowStruct` / `DeleteRowStruct`) to avoid transaction and prepare overhead.

### Transaction nesting avoidance

`Transaction()` detects when the connection is already inside an active transaction and reuses it instead of starting a nested one. This is particularly important for batch operations like `InsertRowStructs` that wrap themselves in a transaction — when called from within an existing transaction, no extra transaction setup or teardown occurs. (Use `IsolatedTransaction()` when a new, independent transaction is needed even within an existing one.)


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

#### Shared test suite (`conntest`)

The `conntest` package provides a shared, driver-agnostic integration test suite. Instead of duplicating tests across every driver, the suite is written once and each driver calls `conntest.RunAll` with a driver-specific `Config`:

```go
func TestConnectionSuite(t *testing.T) {
    conntest.RunAll(t, conntest.Config{
        NewConn:      connectPQ,              // factory that creates a real connection per test
        QueryBuilder: pqconn.QueryBuilder{},  // driver-specific query builder
        DDL: conntest.DDL{
            CreateSimpleTable:    `CREATE TABLE conntest_simple (id INTEGER PRIMARY KEY, val TEXT)`,
            CreateUpsertTable:    `CREATE TABLE conntest_upsert (id INTEGER PRIMARY KEY, name TEXT NOT NULL, score INTEGER NOT NULL DEFAULT 0)`,
            CreateReturningTable: `CREATE TABLE conntest_returning (id SERIAL PRIMARY KEY, name TEXT NOT NULL, score INTEGER NOT NULL DEFAULT 0)`,
        },
        DefaultIsolationLevel:        sql.LevelReadCommitted,
        DriverName:                   pqconn.Driver,
        DatabaseName:                 dbName,
        SupportsReadOnlyTransaction:  true,
        SupportsCustomIsolationLevel: true,
        ExecAfterClosedTxErrors:      true,
    })
}
```

`conntest.Config` captures all vendor differences in one place:

| Field                          | Purpose                                                              |
| ------------------------------ | -------------------------------------------------------------------- |
| `NewConn`                      | Factory that creates a fresh `Connection` for each test              |
| `QueryBuilder`                 | Driver-specific `QueryBuilder` for insert/update/upsert SQL          |
| `DDL`                          | `CREATE TABLE` statements using vendor-specific syntax               |
| `DefaultIsolationLevel`        | Expected default isolation level (e.g. Read Committed for PostgreSQL)|
| `SupportsReadOnlyTransaction`  | Skip read-only transaction tests when not supported                  |
| `SupportsCustomIsolationLevel` | Skip custom isolation level tests when not supported                 |
| `ExecAfterClosedTxErrors`      | Whether executing on a closed transaction returns an error           |

`RunAll` executes the following sub-test groups against the real database:

| Sub-test        | Coverage                                                    |
| --------------- | ----------------------------------------------------------- |
| Basic           | Connection config, ping, `SELECT 1`                         |
| Exec            | INSERT, UPDATE, DELETE, rows affected                       |
| Query           | Single row, multiple rows, scalar values, no-rows handling  |
| Prepare         | Prepared statements                                         |
| Transaction     | Commit, rollback, isolation levels, read-only, savepoints   |
| QueryBuilder    | Struct-based insert, update, delete via query builder       |
| Upsert          | Driver-specific upsert (ON CONFLICT / MERGE / ON DUPLICATE) |
| Returning       | INSERT/UPDATE ... RETURNING (skipped when DDL is empty)     |
| QueryCallback   | Per-row callback queries                                    |
| Batch           | Bulk insert and upsert of struct slices                     |
| MailAddress     | Custom type wrapping with `MailAddressTypeWrapper`          |

Each test gets a fresh connection via `NewConn` and creates/drops its own tables, so tests are fully isolated and safe to run in parallel. Adding a new test to `conntest` automatically covers all drivers.

## History

This package started out as an extension wrapper of [github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx) but turned into a complete rewrite using the same philosophy of representing table rows as Go structs.

It has been used and refined for years in production by [domonda](https://domonda.com) using the database driver [github.com/lib/pq](https://github.com/lib/pq).

The design patterns evolved mostly through discovery led by the desire to minimize boilerplate code while maintaining the full power of SQL.

## License

[MIT](LICENSE)

