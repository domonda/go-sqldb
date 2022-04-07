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
	config := &sqldb.Config{
		Driver: "postgres",
		Host:   "localhost",
		User:   "postgres",
		Extra:  map[string]string{"sslmode": "disable"},
	}

	conn, err := pqconn.New(context.Background(), config)
	if err != nil {
		panic(err)
	}

	ctx := db.ContextWithConn(context.Background(), conn)

	tables, err := information.GetAllTables(ctx)
	if err != nil {
		panic(err)
	}

	pretty.Println(tables, "  ")
}
