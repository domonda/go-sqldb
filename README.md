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

## Testing

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