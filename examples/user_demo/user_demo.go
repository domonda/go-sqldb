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
	sqldb.TableName `db:"public.user"`

	ID uu.ID `db:"id,primarykey,default"`

	Email email.NullableAddress   `db:"email"`
	Title nullable.NonEmptyString `db:"title"`
	Name  string                  `db:"name"`

	SessionToken nullable.NonEmptyString `db:"session_token"`

	CreatedAt  time.Time     `db:"created_at,default"`
	UpdatedAt  time.Time     `db:"updated_at,default"`
	DisabledAt nullable.Time `db:"disabled_at"`
}

func main() {
	ctx := context.Background()

	config := &sqldb.ConnConfig{
		Driver:   "postgres",
		Host:     "localhost",
		User:     "postgres",
		Database: "demo",
		Extra:    map[string]string{"sslmode": "disable"},
	}

	fmt.Println("Connecting to:", config)

	conn, err := pqconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
	if err != nil {
		panic(err)
	}

	// Or use a custom struct reflector
	conn, err = pqconn.ConnectExt(ctx, config, &sqldb.TaggedStructReflector{
		NameTag:          "col",
		Ignore:           "ignore",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	})
	if err != nil {
		panic(err)
	}

	// Query all users as struct slice
	users, err := sqldb.QueryRowsAsSlice[User](ctx, conn, `select * from public.user`)
	if err != nil {
		panic(err)
	}
	fmt.Println(users)

	// Query single column as slice
	userEmails, err := sqldb.QueryRowsAsSlice[string](ctx, conn, `select email from public.user`)
	if err != nil {
		panic(err)
	}
	fmt.Println(userEmails)

	// Callback with scanned values for each row
	err = sqldb.QueryCallback(ctx, conn,
		func(name, email string) {
			fmt.Printf("%q <%s>\n", name, email)
		},
		`select name, email from public.user`,
	)
	if err != nil {
		panic(err)
	}

	// Insert a struct with table name from struct tag
	newUser := &User{ /* ... */ }
	err = sqldb.InsertRowStruct(ctx, conn, newUser)
	if err != nil {
		panic(err)
	}

	// Insert with values map
	err = sqldb.Insert(ctx, conn, "public.user", sqldb.Values{
		"name":  "Erik Unger",
		"email": "erik@domonda.com",
	})
	if err != nil {
		panic(err)
	}

	// Upsert a struct
	err = sqldb.UpsertStruct(ctx, conn, newUser, sqldb.IgnoreColumns("created_at"))
	if err != nil {
		panic(err)
	}

	// Transaction with ConnExt
	txOpts := &sql.TxOptions{Isolation: sql.LevelWriteCommitted}

	err = sqldb.TransactionExt(ctx, conn, txOpts, func(tx *sqldb.ConnExt) error {
		err := tx.Exec(ctx, "...")
		if err != nil {
			return err
		}
		return tx.Exec(ctx, "...")
	})
	if err != nil {
		panic(err)
	}

	// Use context with timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	err = conn.Exec(ctxTimeout, "...")
	if err != nil {
		panic(err)
	}

	// HTTP handler using request context
	_ = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := conn.Exec(request.Context(), "...")
		if err != nil {
			http.Error(response, "internal server error", http.StatusInternalServerError)
			return
		}
		response.Write([]byte("OK"))
	})

	// Full example with db package

	db.SetConn(conn)

	err = db.Exec(ctx, "...")
	if err != nil {
		panic(err)
	}

	userID := uu.IDFrom("b26200df-5973-4ea5-a284-24dd15b6b85b")

	err = db.Transaction(ctx, func(ctx context.Context) error {
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
}

func GetUserOrNil(ctx context.Context, userID uu.ID) (user *User, err error) {
	err = db.QueryRow(ctx,
		`select * from public.user where id = $1`,
		userID,
	).Scan(&user)
	if err != nil {
		return nil, db.ReplaceErrNoRows(err, nil)
	}
	return user, nil
}
