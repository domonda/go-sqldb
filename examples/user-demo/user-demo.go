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

	refl := sqldb.NewTaggedStructReflector()

	config := &sqldb.Config{
		Driver:   "postgres",
		Host:     "localhost",
		User:     "postgres",
		Database: "demo",
		Extra:    map[string]string{"sslmode": "disable"},
	}

	fmt.Println("Connecting to:", config)

	conn, err := pqconn.Connect(ctx, config)
	if err != nil {
		panic(err)
	}

	// Query all users as struct slice
	users, err := sqldb.QueryRowsAsSlice[User](ctx, conn, refl, conn,
		/*sql*/ `SELECT * FROM public.user`,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(users)

	// Query single column as slice
	userEmails, err := sqldb.QueryRowsAsSlice[string](ctx, conn, refl, conn,
		/*sql*/ `SELECT email FROM public.user`,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(userEmails)

	// Callback with scanned values for each row
	err = sqldb.QueryCallback(ctx, conn, refl, conn,
		func(name, email string) {
			fmt.Printf("%q <%s>\n", name, email)
		},
		/*sql*/ `SELECT name, email FROM public.user`,
	)
	if err != nil {
		panic(err)
	}

	// Insert a struct with table name from struct tag
	newUser := &User{ /* ... */ }
	builder := pqconn.QueryBuilder{}
	err = sqldb.InsertRowStruct(ctx, conn, refl, builder, conn, newUser)
	if err != nil {
		panic(err)
	}

	// Insert with values map
	err = sqldb.Insert(ctx, conn, builder, conn, "public.user", sqldb.Values{
		"name":  "Erik Unger",
		"email": "erik@domonda.com",
	})
	if err != nil {
		panic(err)
	}

	// Upsert a struct
	err = sqldb.UpsertRowStruct(ctx, conn, refl, builder, conn, newUser, sqldb.IgnoreColumns("created_at"))
	if err != nil {
		panic(err)
	}

	// Transaction
	txOpts := &sql.TxOptions{Isolation: sql.LevelWriteCommitted}

	err = sqldb.Transaction(ctx, conn, txOpts, func(tx sqldb.Connection) error {
		err := tx.Exec(ctx,
			/*sql*/ `UPDATE public.user SET disabled_at = now() WHERE id = $1`,
			newUser.ID,
		)
		if err != nil {
			return err
		}
		return tx.Exec(ctx,
			/*sql*/ `UPDATE public.user SET session_token = NULL WHERE id = $1`,
			newUser.ID,
		)
	})
	if err != nil {
		panic(err)
	}

	// Use context with timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	err = conn.Exec(ctxTimeout,
		/*sql*/ `DELETE FROM public.user WHERE disabled_at < now() - interval '1 year'`,
	)
	if err != nil {
		panic(err)
	}

	// HTTP handler using request context
	_ = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := conn.Exec(request.Context(),
			/*sql*/ `UPDATE public.user SET session_token = $1 WHERE id = $2`,
			request.Header.Get("X-Session-Token"),
			request.Header.Get("X-User-ID"),
		)
		if err != nil {
			http.Error(response, "internal server error", http.StatusInternalServerError)
			return
		}
		_, _ = response.Write([]byte("OK"))
	})

	// Full example with db package

	db.SetConn(conn)

	userID := uu.IDFrom("b26200df-5973-4ea5-a284-24dd15b6b85b")

	err = db.Exec(ctx,
		/*sql*/ `UPDATE public.user SET updated_at = now() WHERE id = $1`,
		userID,
	)
	if err != nil {
		panic(err)
	}

	err = db.Transaction(ctx, func(ctx context.Context) error {
		user, err := GetUserOrNil(ctx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return db.Exec(ctx,
				/*sql*/ `INSERT INTO public.user (id, name) VALUES ($1, 'New User')`,
				userID,
			)
		}
		return db.Exec(ctx,
			/*sql*/ `UPDATE public.user SET name = $1 WHERE id = $2`,
			user.Name+" (updated)",
			userID,
		)
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
