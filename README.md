# go-sqldb

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-sqldb.svg)](https://pkg.go.dev/github.com/domonda/go-sqldb) [![license](https://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/domonda/go-sqldb/master/LICENSE)

This package started out as an extension wrapper of [github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx) but turned into a complete rewrite using the same philisophy of representing table rows as Go structs.

It has been used and refined for years in production by [domonda](https://domonda.com) using the database driver [github.com/lib/pq](https://github.com/lib/pq).

The design patters evolved mostly through discovery lead by the deisire tominimize boilerplate code while maintaining the full power of SQL. 

## Philosopy

* Use reflection to map db rows to structs, but not as full blown ORM that replaces SQL queries (just as much ORM to increase productivity but not alienate developers who like the full power of SQL)
* Transactions are run in callback functions that can be nested
* Option to store the db connection and transactions in the context argument to pass it down into nested functions


### Database drivers

* [pqconn](https://pkg.go.dev/github.com/domonda/go-sqldb/pqconn) using [github.com/lib/pq](https://github.com/lib/pq) 
* [mysqlconn](https://pkg.go.dev/github.com/domonda/go-sqldb/mysqlconn) using [github.com/go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)


### Creating a connection

The connection is pinged with the passed context
and only returned when there was no error from the ping:

```go
config := &sqldb.Config{
    Driver:   "postgres",
    Host:     "localhost",
    User:     "postgres",
    Database: "demo",
    Extra:    map[string]string{"sslmode": "disable"},
}

fmt.Println("Connecting to:", config.ConnectURL())

conn, err := pqconn.New(context.Background(), config)
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
var users []User
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
