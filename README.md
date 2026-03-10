# go-sqldb

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-sqldb.svg)](https://pkg.go.dev/github.com/domonda/go-sqldb) [![license](https://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/domonda/go-sqldb/master/LICENSE)

This package started out as an extension wrapper of [github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx) but turned into a complete rewrite using the same philosophy of representing table rows as Go structs.

It has been used and refined for years in production by [domonda](https://domonda.com) using the database driver [github.com/lib/pq](https://github.com/lib/pq).

The design patterns evolved mostly through discovery led by the desire to minimize boilerplate code while maintaining the full power of SQL.

## Philosophy

* Use reflection to map db rows to structs, but not as full blown ORM that replaces SQL queries (just as much ORM to increase productivity but not alienate developers who like the full power of SQL)
* Transactions are run in callback functions that can be nested
* Option to store the db connection and transactions in the context argument to pass it down into nested functions


## Database drivers

* [pqconn](https://pkg.go.dev/github.com/domonda/go-sqldb/pqconn) using [github.com/lib/pq](https://github.com/lib/pq)
* [mysqlconn](https://pkg.go.dev/github.com/domonda/go-sqldb/mysqlconn) using [github.com/go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
* [mssqlconn](https://pkg.go.dev/github.com/domonda/go-sqldb/mssqlconn) using [github.com/microsoft/go-mssqldb](https://github.com/microsoft/go-mssqldb)
* [sqliteconn](https://pkg.go.dev/github.com/domonda/go-sqldb/sqliteconn) using [zombiezen.com/go/sqlite](https://pkg.go.dev/zombiezen.com/go/sqlite)


## Generic Errors

Each driver maps its database-specific constraint errors to typed values defined in the root `sqldb` package:

| Type                              | `Constraint` field | Description                                   |
| --------------------------------- | ------------------ | --------------------------------------------- |
| `ErrIntegrityConstraintViolation` | constraint name    | Base type for all constraint violations       |
| `ErrNotNullViolation`             | column name        | NULL inserted into a NOT NULL column          |
| `ErrUniqueViolation`              | index/constraint   | Duplicate value for a unique key              |
| `ErrForeignKeyViolation`          | constraint name    | Referential integrity violation               |
| `ErrCheckViolation`               | constraint name    | CHECK constraint violated                     |
| `ErrRestrictViolation`            | constraint name    | RESTRICT constraint violated (PostgreSQL)     |
| `ErrExclusionViolation`           | constraint name    | Exclusion constraint violated (PostgreSQL)    |

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

Driver packages also expose driver-specific helper functions (e.g. `pqconn.IsUniqueViolation`) for error conditions that have no generic `sqldb` type, such as deadlocks, query cancellations, or text-representation errors. See each driver's README for the full list.


## Usage

### Creating a connection

The connection is pinged with the passed context
and only returned when there was no error from the ping:

```go
config := &sqldb.ConnConfig{
    Driver:   "postgres",
    Host:     "localhost",
    User:     "postgres",
    Database: "demo",
    Extra:    map[string]string{"sslmode": "disable"},
}

fmt.Println("Connecting to:", config.String())

conn, err := pqconn.Connect(context.Background(), config)
```

### Struct field mapping

Every connection uses a `StructReflector` to map struct fields to database columns.
The default reflector uses the `db` struct tag:

```go
type User struct {
    ID    uu.ID  `db:"id,primarykey"`
    Email string `db:"email"`
    Name  string `db:"name"`
    // Field without tag or with tag "-" will be ignored
    Internal string `db:"-"`
}
```

You can customize the struct reflector when creating a connection:

```go
// Create a custom reflector
reflector := &sqldb.TaggedStructReflector{
    NameTag:          "col",           // Use "col" tag instead of "db"
    Ignore:           "_ignore_",      // Ignore fields with this value
    PrimaryKey:       "pk",
    ReadOnly:         "readonly",
    Default:          "default",
    UntaggedNameFunc: sqldb.ToSnakeCase, // Convert untagged fields to snake_case
}

// Create ConnExt with custom reflector
connExt := sqldb.NewConnExt(
    conn,
    reflector,
    sqldb.StdQueryFormatter{},
    sqldb.StdQueryBuilder{},
)
```

### Slice and array column handling

Slice and array column handling (like PostgreSQL arrays) is handled transparently by vendor connection implementations, not the base `sqldb` package. For example, the `pqconn` driver automatically wraps Go slices and arrays with `pq.Array()` for both query arguments (input) and row scanning (output), so you can use native Go slices in structs mapped to PostgreSQL array columns without any manual conversion.

### Exec SQL without reading rows

```go
err = conn.Exec(`delete from public.user where id = $1`, userID)
```

### Single row query

```go
type User struct {
	ID    uu.ID  `db:"id,pk"`
	Email string `db:"email"`
	Name  string `db:"name"`
}

var user User
err = conn.QueryRow(`select * from public.user where id = $1`, userID).ScanStruct(&user)

var userExists bool
err = conn.QueryRow(`select exists(select from public.user where email = $1)`, userEmail).Scan(&userExists)
```

### Multi rows query

```go
var users []*User
err = conn.QueryRows(`select * from public.user`).ScanStructSlice(&users)

var userEmails []string
err = conn.QueryRows(`select email from public.user`).ScanSlice(&userEmails)

// Use reflection for callback function arguments
err = conn.QueryRows(`select name, email from public.user`).ForEachRowCall(
    func(name, email string) {
        fmt.Printf("%q <%s>\n", name, email)
    },
)

err = conn.QueryRows(`select name, email from public.user`).ForEachRow(
    func(row sqldb.RowScanner) error {
        var name, email string
        err := row.Scan(&name, &email)
        if err != nil {
            return err
        }
        _, err = fmt.Printf("%q <%s>\n", name, email)
        return err
    },
)
```

### Insert rows

```go
newUser := &User{ /* ... */ }

err = conn.InsertStruct("public.user", newUser)

// Use column defaults for insert instead of struct fields
err = conn.InsertStructIgnoreColumns("public.user", newUser, "id", "created_at")

// Upsert uses columns marked as primary key like `db:"id,pk"`
err = conn.UpsertStructIgnoreColumns("public.user", newUser, "created_at")

// Without structs
err = conn.Insert("public.user", sqldb.Values{
    "name":  "Erik Unger",
    "email": "erik@domonda.com",
})
```

### Transactions

```go
txOpts := &sql.TxOptions{Isolation: sql.LevelWriteCommitted}

err = sqldb.Transaction(conn, txOpts, func(tx sqldb.Connection) error {
    err := tx.Exec("...")
    if err != nil {
        return err // roll back tx
    }
    return tx.Exec("...")
})
```

### Using the context

Saving a context in a struct is an antipattern in Go
but it turns out that it allows neat call chaining pattern.

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
defer cancel()

// Note that this timout is a deadline and does not restart for every query
err = conn.WithContext(ctx).Exec("...")

// Usually the context comes from some top-level handler
_ = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
    // Pass request cancellation through to db query
    err := conn.WithContext(request.Context()).Exec("...")
    if err != nil {
        http.Error(response, err.Error(), http.StatusInternalServerError)
        return
    }
    response.Write([]byte("OK"))
})
```

### Putting it all together with the db package

The [github.com/domonda/go-sqldb/db](https://pkg.go.dev/github.com/domonda/go-sqldb/db)
package enables a design pattern where a "current" db connection or transaction
can be stored in the context and then retrieved by nested functions
from the context without having to know if this connection is a transaction or not.
This allows re-using the same functions within transactions or standalone.

```go
// Configure the global parent connection
db.SetConn(conn)

// db.Conn(ctx) is the standard pattern
// to retrieve a connection anywhere in the code base
err = db.Conn(ctx).Exec("...")
```

Here if `GetUserOrNil` will use the global db connection if
no other connection is stored in the context.

But when called from within the function passed to `db.Transaction`
it will re-use the transaction saved in the context.


```go
func GetUserOrNil(ctx context.Context, userID uu.ID) (user *User, err error) {
	err = db.Conn(ctx).QueryRow(
		`select * from public.user where id = $1`,
		userID,
	).ScanStruct(&user)
	if err != nil {
		return nil, db.ReplaceErrNoRows(err, nil)
	}
	return user, nil
}

func DoStuffWithinTransation(ctx context.Context, userID uu.ID) error {
	return db.Transaction(ctx, func(ctx context.Context) error {
		user, err := GetUserOrNil(ctx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return db.Conn(ctx).Exec("...")
		}
		return db.Conn(ctx).Exec("...")
	})
}
```

Small helpers:

```go
err = db.TransactionOpts(ctx, &sql.TxOptions{ReadOnly: true}, func(context.Context) error { ... })

err = db.TransactionReadOnly(ctx, func(context.Context) error { ... })

// Execute the passed function without transaction
err = db.DebugNoTransaction(ctx, func(context.Context) error { ... })
```

More sophisticated transactions:

Serialized transactions are typically necessary when an insert depends on a previous select within
the transaction, but that pre-insert select can't lock the table like it's possible with `SELECT FOR UPDATE`.
```go
err = db.SerializedTransaction(ctx, func(context.Context) error { ... })
```

`TransactionSavepoint` executes `txFunc` within a database transaction or uses savepoints for rollback.
If the passed context already has a database transaction connection,
then a uniquely named savepoint (sp1, sp2, ...) is created before the execution of `txFunc`.
If `txFunc` returns an error, then the transaction is rolled back to the savepoint
but the transaction from the context is not rolled back.
If the passed context does not have a database transaction connection,
then `Transaction(ctx, txFunc)` is called without savepoints.
```go
err = db.TransactionSavepoint(ctx, func(context.Context) error { ... })
```

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

`MockConn` implements the `ListenerConnection` interface entirely in memory, allowing you to unit test database-dependent code without a running database.

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

Wrap the `MockConn` in a `ConnExt` and set it as the connection in the context or globally:

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

    connExt := sqldb.NewConnExt(
        mockConn,
        new(sqldb.TaggedStructReflector), // default struct reflector
        mockConn.QueryFormatter,          // reuse the formatter from mockConn
        sqldb.StdQueryBuilder{},
    )

    ctx := db.ContextWithConn(t.Context(), connExt)

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

For dynamic query responses, set the `MockQuery` function instead of using `MockQueryResults`:

```go
mockConn.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
    if strings.Contains(query, "public.user") {
        return sqldb.NewMockRows("id", "name").
            WithRow("some-id", "Alice")
    }
    return sqldb.NewErrRows(sql.ErrNoRows)
}
```

Note: when `MockQuery` is set, `MockQueryResults` is not consulted.

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

    connExt := sqldb.NewConnExt(
        mockConn,
        new(sqldb.TaggedStructReflector),
        mockConn.QueryFormatter,
        sqldb.StdQueryBuilder{},
    )
    ctx := db.ContextWithConn(t.Context(), connExt)

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

Integration tests use a dockerized PostgreSQL 17 instance on port 5433 (to avoid conflicts with a local PostgreSQL on the default port 5432).

Start the test database:
```bash
docker compose -f pqconn/test/docker-compose.yml up -d
```

Run all tests:
```bash
./test-workspace.sh
```

### Changing the PostgreSQL version

After changing the PostgreSQL image version in `pqconn/test/docker-compose.yml`, the data directory must be reset because PostgreSQL data files are not compatible across major versions:

```bash
./pqconn/test/reset-postgres-data.sh
```