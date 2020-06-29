package main

import (
	"context"

	"github.com/domonda/go-pretty"
	"github.com/domonda/go-sqldb"
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

	db := information.NewDatabase(conn)

	tables, err := db.GetTables()
	if err != nil {
		panic(err)
	}

	pretty.Println(tables, "  ")
}
