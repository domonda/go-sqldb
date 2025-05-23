package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
	"github.com/domonda/go-sqldb/pqconn"
	"github.com/domonda/go-types/email"
	"github.com/domonda/go-types/nullable"
	"github.com/domonda/go-types/uu"
)

type User struct {
	ID uu.ID `db:"id,pk,default"`

	Email email.NullableAddress   `db:"email"`
	Title nullable.NonEmptyString `db:"title"`
	Name  string                  `db:"name"`

	SessionToken nullable.NonEmptyString `db:"session_token"`

	CreatedAt  time.Time     `db:"created_at,default"`
	UpdatedAt  time.Time     `db:"updated_at,default"`
	DisabledAt nullable.Time `db:"disabled_at"`
}

func setupDB() {
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

	conn = conn.WithStructFieldMapper(&sqldb.TaggedStructFieldMapping{
		NameTag:          "col",
		Ignore:           "ignore",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	})

	db.SetConn(conn)
}

func main() {
	setupDB()

	ctx := context.Background()

	var users []User
	err := db.QueryRows(ctx, `select * from public.user`).ScanStructSlice(&users)
	if err != nil {
		panic(err)
	}

	var userEmails []string
	err = db.QueryRows(ctx, `select email from public.user`).ScanSlice(&userEmails)
	if err != nil {
		panic(err)
	}

	err = db.QueryRows(ctx, `select name, email from public.user`).ForEachRow(
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

	err = db.QueryRows(ctx, `select name, email from public.user`).ForEachRowCall(
		func(name, email string) {
			fmt.Printf("%q <%s>\n", name, email)
		},
	)
	if err != nil {
		panic(err)
	}

	newUser := &User{ /* ... */ }
	err = db.InsertStruct(ctx, "public.user", newUser)
	if err != nil {
		panic(err)
	}

	err = db.InsertStruct(ctx, "public.user", newUser, sqldb.IgnoreNullOrZeroDefault)
	if err != nil {
		panic(err)
	}

	err = db.Insert(ctx, "public.user", sqldb.Values{
		"name":  "Erik Unger",
		"email": "erik@domonda.com",
	})
	if err != nil {
		panic(err)
	}

	err = db.UpsertStruct(ctx, "public.user", newUser, sqldb.IgnoreColumns("created_at"))
	if err != nil {
		panic(err)
	}

	userID := uu.IDFrom("b26200df-5973-4ea5-a284-24dd15b6b85b")

	txOpts := &sql.TxOptions{Isolation: sql.LevelWriteCommitted}
	err = db.TransactionOpts(ctx, txOpts, func(ctx context.Context) error {
		user, err := GetUserOrNil(ctx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return db.Exec(ctx, "...")
		}
		return db.Exec(ctx, "...")
	})
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = db.Exec(ctx, "...")
	if err != nil {
		panic(err)
	}

	_ = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := db.Exec(request.Context(), "...")
		if err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		response.Write([]byte("OK"))
	})

}

func GetUserOrNil(ctx context.Context, userID uu.ID) (user *User, err error) {
	err = db.QueryRow(ctx,
		`select * from public.user where id = $1`,
		userID,
	).ScanStruct(&user)
	if err != nil {
		return nil, db.ReplaceErrNoRows(err, nil)
	}
	return user, nil
}
