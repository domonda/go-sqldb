package db_test

import (
	"context"
	"os"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

func ExampleUpsertStructStmt() {
	type User struct {
		sqldb.TableName `db:"public.user"`
		ID              int64  `db:"id,primarykey"`
		Name            string `db:"name"`
		Email           string `db:"email"`
	}

	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
	}

	conn := &sqldb.MockConn{
		QueryFormatter: sqldb.NewQueryFormatter("$"),
		QueryLog:       os.Stdout,
	}
	ctx := db.ContextWithConn(context.Background(), conn)

	err := db.Transaction(ctx, func(ctx context.Context) error {
		upsert, done, err := db.UpsertStructStmt[User](ctx)
		if err != nil {
			return err
		}
		defer done()

		for _, user := range users {
			err = upsert(ctx, user)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Output:
	// BEGIN;
	// PREPARE stmt1 AS INSERT INTO public.user(id,name,email) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, email=$3;
	// INSERT INTO public.user(id,name,email) VALUES(1,'Alice','alice@example.com') ON CONFLICT(id) DO UPDATE SET name='Alice', email='alice@example.com';
	// INSERT INTO public.user(id,name,email) VALUES(2,'Bob','bob@example.com') ON CONFLICT(id) DO UPDATE SET name='Bob', email='bob@example.com';
	// INSERT INTO public.user(id,name,email) VALUES(3,'Charlie','charlie@example.com') ON CONFLICT(id) DO UPDATE SET name='Charlie', email='charlie@example.com';
	// DEALLOCATE PREPARE stmt1;
	// COMMIT;
}
