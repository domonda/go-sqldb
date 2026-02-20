package main

import (
	"context"

	"github.com/domonda/go-pretty"
	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
	"github.com/domonda/go-sqldb/information"
	"github.com/domonda/go-sqldb/pqconn"
)

func main() {
	config := &sqldb.ConnConfig{
		Driver: "postgres",
		Host:   "localhost",
		User:   "postgres",
		Extra:  map[string]string{"sslmode": "disable"},
	}

	ctx := context.Background()

	conn, err := pqconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
	if err != nil {
		panic(err)
	}

	ctx = db.ContextWithConn(ctx, conn)

	tables, err := information.GetAllTables(ctx)
	if err != nil {
		panic(err)
	}

	pretty.Println(tables, "  ")
}
