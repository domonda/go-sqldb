package db_test

import (
	"time"

	"github.com/domonda/go-sqldb/db"
)

func ExampleUpsertStructStmt() {
	type User struct {
		db.TableName `db:"public.user"`
		ID           int64     `db:"id,primarykey,default"`
		Name         string    `db:"name"`
		Email        string    `db:"email"`
		CreatedAt    time.Time `db:"created_at,default"`
	}

	// Output:

}
