# db

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-sqldb/db.svg)](https://pkg.go.dev/github.com/domonda/go-sqldb/db)

Package `db` provides high-level database operations using a global connection with context-based overrides. It is the recommended entry point for working with `go-sqldb`.

## Connection resolution

Every function in this package takes a `context.Context` as its first argument and resolves the database connection in this order:

1. **Context connection** — set via `ContextWithConn(ctx, conn)`, e.g. a transaction or mock connection
2. **Global connection** — set once at startup via `SetConn(conn)`

This means business logic never needs to accept a connection parameter explicitly. Functions just call `db.QueryRowAs[T](ctx, ...)` and the right connection is used automatically — whether it's the global connection, a transaction, or a mock.

### Startup

Using the PostgreSQL pqconn driver:

```go
conn, err := pqconn.Connect(ctx, config)
if err != nil {
    panic(err)
}
defer conn.Close()

db.SetConn(conn)
```

### Struct-based operations

Define a struct with an embedded `db.TableName` to specify the table and use `db` struct tags to map fields to columns. Mark primary key columns with `,primarykey` and columns that have database defaults with `,default`:

```go
type User struct {
    db.TableName `db:"public.user"`

    ID        uu.ID     `db:"id,primarykey"`
    Email     string    `db:"email"`
    Name      string    `db:"name"`
    CreatedAt time.Time `db:"created_at,default"`
}
```

Insert a row and then query it back by primary key:

```go
userID := uu.IDv4()

err := db.InsertRowStruct(ctx, 
    &User{
        ID:    userID,
        Email: "alice@example.com",
        Name:  "Alice",
    },
    db.IgnoreHasDefault, // prevents the zero CreatedAt field to be inserted
)
if err != nil {
    return err
}

// user.CreatedAt will contain the default value created for the new row
user, err := db.QueryRowStruct[User](ctx, userID)
if err != nil {
    return err
}
```

### Transactions

`db.Transaction` wraps a callback in a database transaction. Inside the callback, `ctx` carries the transaction connection so all `db.*` calls use it transparently:

```go
err := db.Transaction(ctx, func(ctx context.Context) error {
    user, err := db.QueryRowAs[User](ctx, `SELECT * FROM public.user WHERE id = $1`, id)
    if err != nil {
        return err
    }
    return db.Exec(ctx, `UPDATE public.user SET last_login = now() WHERE id = $1`, id)
})
```

Nested `Transaction` calls reuse the parent transaction (no additional BEGIN/COMMIT). Use `IsolatedTransaction` to force a new transaction even when already inside one.

### Multi-column queries

When a query returns a fixed number of scalar columns, use `QueryRowAs2` through `QueryRowAs5` to scan them into separate typed variables without defining a struct:

```go
id, name, active, err := db.QueryRowAs3[int, string, bool](ctx,
    `SELECT id, name, active FROM public.user WHERE id = $1`, userID)
if err != nil {
    return err
}
```

### Query builder and struct reflector

Besides the connection, the `db` package manages two more components with the same global-plus-context pattern:

- **QueryBuilder** generates SQL for struct-based operations (insert, update, upsert, query by primary key). The default `StdReturningQueryBuilder` produces standard SQL for CRUD and RETURNING operations. For upsert and insert-unique operations, the builder must also implement `UpsertQueryBuilder` — driver connections provide this automatically (e.g. `pqconn`, `mysqlconn`). Resolution order:
  1. Context value via `ContextWithQueryBuilder`
  2. The connection itself, if it implements `QueryBuilder`
  3. Global value set via `SetQueryBuilder`

- **StructReflector** maps Go struct fields to database columns using struct tags. The default `TaggedStructReflector` uses the `db` tag (e.g. `db:"column_name,primarykey"`). Resolution order:
  1. Context value via `ContextWithStructReflector`
  2. Global value set via `SetStructReflector`

Most applications never need to change these defaults. Override them when you need custom SQL generation (e.g. for a non-standard SQL dialect) or a different struct-to-column mapping strategy:

```go
// Use a different struct tag for column mapping
reflector := &sqldb.TaggedStructReflector{
    NameTag:          "col",
    Ignore:           "-",
    PrimaryKey:       "pk",
    ReadOnly:         "readonly",
    Default:          "default",
    UntaggedNameFunc: sqldb.ToSnakeCase,
}
db.SetStructReflector(reflector)

// Or override per request via context
ctx = db.ContextWithStructReflector(ctx, reflector)
```

### Mock connections for tests

Inject a `MockConn` via the context to unit-test database code without a running database:

```go
func TestGetUser(t *testing.T) {
    mockConn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
        WithQueryResult(
            []string{"id", "email", "name"}, // columns
            [][]driver.Value{{"some-id", "alice@example.com", "Alice"}}, // rows
            `SELECT * FROM public.user WHERE id = $1`, // query
            "some-id", // args
        )

    ctx := db.ContextWithConn(t.Context(), mockConn)

    user, err := GetUser(ctx, "some-id")
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

## Function reference

### Setup and connection management

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `SetConn(conn)`                          | Set the global connection                |
| `Conn(ctx) Connection`                   | Get the connection from context or global |
| `ContextWithConn(ctx, conn) context.Context` | Override connection in context           |
| `ContextWithGlobalConn(ctx) context.Context` | Explicitly use the global connection in context |
| `Close() error`                          | Close the global connection              |
| `SetQueryBuilder(qb)`                    | Set the global query builder             |
| `QueryBuilder(ctx) QueryBuilder`         | Get the query builder from context, connection, or global |
| `ContextWithQueryBuilder(ctx, qb) context.Context` | Override query builder in context        |
| `SetStructReflector(sr)`                 | Set the global struct reflector          |
| `StructReflector(ctx) StructReflector`   | Get the struct reflector from context or global |
| `ContextWithStructReflector(ctx, sr) context.Context` | Override struct reflector in context     |

### Query — single row

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `QueryRow(ctx, query, args...) *Row`     | Query a single row for manual `Scan`     |
| `QueryRowAs[T](ctx, query, args...) (T, error)` | Query a single row into a value or struct |
| `QueryRowAs2[T0,T1](ctx, query, args...) (T0, T1, error)` | Query a single row into 2 typed values |
| `QueryRowAs3[T0,T1,T2](ctx, query, args...) (T0, T1, T2, error)` | Query a single row into 3 typed values |
| `QueryRowAs4[T0,T1,T2,T3](ctx, query, args...) (T0, T1, T2, T3, error)` | Query a single row into 4 typed values |
| `QueryRowAs5[T0,T1,T2,T3,T4](ctx, query, args...) (T0, T1, T2, T3, T4, error)` | Query a single row into 5 typed values |
| `QueryRowAsOr[T](ctx, defaultVal, query, args...) (T, error)` | Like `QueryRowAs` but returns `defaultVal` instead of `ErrNoRows` |
| `QueryRowAsStmt[T](ctx, query) (func, closeStmt, error)` | Prepared statement returning a reusable query function |
| `QueryRowStruct[S](ctx, pkValue, pkValues...) (S, error)` | Query a struct by primary key            |
| `QueryRowStructOr[S](ctx, defaultVal, pkValue, pkValues...) (S, error)` | Like `QueryRowStruct` but returns `defaultVal` instead of `ErrNoRows` |
| `QueryRowAsMap[K, V](ctx, query, args...) (map[K]V, error)` | Query a single row into a map            |

### Query — multiple rows

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `QueryRowsAsSlice[T](ctx, query, args...) ([]T, error)` | Query rows into a slice of values or structs |
| `QueryRowsAsStrings(ctx, query, args...) ([][]string, error)` | Query rows as string slices              |
| `QueryCallback(ctx, callback, query, args...) error` | Call a function for each row             |
| `QueryStructCallback[S](ctx, callback, query, args...) error` | Call a function for each row scanned into a struct |

### Exec

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `Exec(ctx, query, args...) error`        | Execute a query without returning rows   |
| `ExecRowsAffected(ctx, query, args...) (int64, error)` | Execute a query and return number of rows affected |
| `ExecStmt(ctx, query) (func, closeStmt, error)` | Prepared statement returning a reusable exec function |
| `ExecRowsAffectedStmt(ctx, query) (func, closeStmt, error)` | Prepared statement returning a reusable exec-rows-affected function |

### Insert

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `Insert(ctx, table, values) error`       | Insert a row from a values map           |
| `InsertUnique(ctx, table, values, onConflict) (bool, error)` | Insert with conflict handling, returns whether a row was inserted |
| `InsertReturning(ctx, table, values, returning) *Row` | Insert with a RETURNING clause           |
| `InsertRowStruct(ctx, rowStruct, options...) error` | Insert a struct                          |
| `InsertRowStructStmt[S](ctx, options...) (func, closeStmt, error)` | Prepared statement for inserting structs |
| `InsertUniqueRowStruct(ctx, rowStruct, onConflict, options...) (bool, error)` | Insert a struct with conflict handling   |
| `InsertRowStructs[S](ctx, rowStructs, options...) error` | Batch insert a slice of structs          |

### Update

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `Update(ctx, table, values, where, args...) error` | Update rows using a values map and WHERE clause |
| `UpdateRowStruct(ctx, rowStruct, options...) error` | Update a row from a struct (WHERE from primary key) |
| `UpdateRowStructStmt[S](ctx, options...) (func, closeStmt, error)` | Prepared statement for updating structs  |
| `UpdateRowStructs[S](ctx, rowStructs, options...) error` | Batch update a slice of structs          |

### Delete

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `DeleteRowStruct(ctx, rowStruct) error`  | Delete a row matching a struct's primary key; returns wrapped `sql.ErrNoRows` if no row affected |
| `DeleteRowStructStmt[S](ctx) (func, closeStmt, error)` | Prepared statement for deleting structs; deleteFunc returns wrapped `sql.ErrNoRows` if no row affected |
| `DeleteRowStructs[S](ctx, rowStructs) error` | Batch delete a slice of structs; returns wrapped `sql.ErrNoRows` if any struct has no matching row |

### Upsert

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `UpsertRowStruct(ctx, rowStruct, options...) error` | Insert or update a struct on primary key conflict |
| `UpsertRowStructStmt[S](ctx, options...) (func, closeStmt, error)` | Prepared statement for upserting structs |
| `UpsertRowStructs[S](ctx, rowStructs, options...) error` | Batch upsert a slice of structs          |

### Transactions

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `Transaction(ctx, txFunc) error`         | Execute within a transaction (reuses parent if nested) |
| `TransactionResult[T](ctx, txFunc) (T, error)` | Transaction returning a value            |
| `TransactionOpts(ctx, opts, txFunc) error` | Transaction with `sql.TxOptions`         |
| `TransactionOptsResult[T](ctx, opts, txFunc) (T, error)` | Transaction with options returning a value |
| `TransactionReadOnly(ctx, txFunc) error` | Read-only transaction                    |
| `TransactionReadOnlyResult[T](ctx, txFunc) (T, error)` | Read-only transaction returning a value  |
| `IsolatedTransaction(ctx, txFunc) error` | Always starts a new transaction, even if already in one |
| `IsolatedTransactionResult[T](ctx, txFunc) (T, error)` | Isolated transaction returning a value   |
| `SerializedTransaction(ctx, txFunc) error` | Serializable isolation with automatic retry |
| `SerializedTransactionResult[T](ctx, txFunc) (T, error)` | Serialized transaction returning a value |
| `TransactionSavepoint(ctx, txFunc) error` | Savepoint for partial rollback within a transaction |
| `TransactionSavepointResult[T](ctx, txFunc) (T, error)` | Savepoint transaction returning a value  |
| `OptionalTransaction(ctx, useTransaction, txFunc) error` | Conditionally wrap in a transaction      |
| `OptionalTransactionResult[T](ctx, useTransaction, txFunc) (T, error)` | Conditional transaction returning a value |
| `DebugNoTransaction(ctx, txFunc) error`  | Execute without transaction (for debugging) |
| `DebugNoTransactionResult[T](ctx, txFunc) (T, error)` | No-transaction debug returning a value   |

### Transaction utilities

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `IsTransaction(ctx) bool`                | Check if context carries a transaction   |
| `ValidateWithinTransaction(ctx) error`   | Return error if not in a transaction     |
| `ValidateNotWithinTransaction(ctx) error` | Return error if in a transaction         |
| `ContextWithoutTransactions(ctx) context.Context` | Disable transaction handling for this context |
| `IsContextWithoutTransactions(ctx) bool` | Check if transactions are disabled       |
| `ContextWithSavepointFunc(ctx, func) context.Context` | Inject custom savepoint naming           |

### Listen/Notify

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `ListenOnChannel(ctx, channel, onNotify, onUnlisten) error` | Subscribe to PostgreSQL NOTIFY           |
| `UnlistenChannel(ctx, channel) error`    | Unsubscribe from a channel               |
| `IsListeningOnChannel(ctx, channel) bool` | Check if listening on a channel          |

### Errors

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `ReplaceErrNoRows(err, replacement) error` | Replace `sql.ErrNoRows` with a custom error or nil |
| `IsOtherThanErrNoRows(err) bool`         | Check if an error is something other than `ErrNoRows` |

### Other

| Function                                 | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| `CurrentTimestamp(ctx) time.Time`        | Get the current database timestamp       |
| `Prepare(ctx, query) (Stmt, error)`      | Prepare a statement                      |
| `NewMockConn(ctx) *MockConn`             | Create a MockConn using the context's query formatter |
| `NewMockStructRows[S](ctx, rows...) *MockStructRows[S]` | Create mock rows from structs            |
