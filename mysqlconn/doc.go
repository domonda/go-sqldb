/*
Package mysqlconn implements github.com/domonda/go-sqldb.Connection
for MySQL and MariaDB using github.com/go-sql-driver/mysql.

Basic usage:

	import (
		"context"
		"github.com/domonda/go-sqldb"
		"github.com/domonda/go-sqldb/mysqlconn"
	)

	config := &sqldb.ConnConfig{
		Driver:   mysqlconn.Driver,
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "mydb",
	}

	conn, err := mysqlconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

The connection uses ? placeholders and backtick identifier quoting.

MySQL/MariaDB-specific features:
  - Default isolation level is sql.LevelRepeatableRead
  - EscapeIdentifier wraps identifiers in backticks when needed
  - DropAllTables disables foreign key checks to drop tables in any order
*/
package mysqlconn
