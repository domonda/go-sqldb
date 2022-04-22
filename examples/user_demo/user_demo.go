package main

import (
	"context"
	"fmt"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/pqconn"
	"github.com/domonda/go-types/email"
	"github.com/domonda/go-types/nullable"
	"github.com/domonda/go-types/uu"
)

type User struct {
	ID uu.ID `db:"id,pk"`

	Email email.NullableAddress   `db:"email"`
	Title nullable.NonEmptyString `db:"title"`
	Name  string                  `db:"name"`

	SessionToken nullable.NonEmptyString `db:"session_token"`

	CreatedAt  time.Time     `db:"created_at"`
	UpdatedAt  time.Time     `db:"updated_at"`
	DisabledAt nullable.Time `db:"disabled_at"`
}

func main() {
	config := &sqldb.Config{
		Driver:   "postgres",
		Host:     "localhost",
		User:     "postgres",
		Database: "demo",
		Extra:    map[string]string{"sslmode": "disable"},
	}

	fmt.Println("Connecting to:", config.ConnectURL())

	conn, err := pqconn.New(context.Background(), config)
	if err != nil {
		panic(err)
	}

	var users []User
	err = conn.QueryRows(`select * from public.user`).ScanStructSlice(&users)
	if err != nil {
		panic(err)
	}

	var userEmails []string
	err = conn.QueryRows(`select email from public.user`).ScanSlice(&userEmails)
	if err != nil {
		panic(err)
	}

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
	if err != nil {
		panic(err)
	}

	err = conn.QueryRows(`select name, email from public.user`).ForEachRowCall(
		func(name, email string) {
			fmt.Printf("%q <%s>\n", name, email)
		},
	)
	if err != nil {
		panic(err)
	}

	newUser := &User{ /* ... */ }
	err = conn.InsertStruct("public.user", newUser)
	if err != nil {
		panic(err)
	}

	err = conn.InsertStructIgnoreColumns("public.user", newUser, "created_at", "updated_at")
	if err != nil {
		panic(err)
	}

	err = conn.Insert("public.user", sqldb.Values{
		"name":  "Erik Unger",
		"email": "erik@domonda.com",
	})
	if err != nil {
		panic(err)
	}

	err = conn.UpsertStructIgnoreColumns("public.user", newUser, "created_at")
	if err != nil {
		panic(err)
	}
}
