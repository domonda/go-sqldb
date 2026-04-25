module github.com/domonda/go-sqldb/information/mysql_information_test

go 1.24.6

replace (
	github.com/domonda/go-sqldb => ../..
	github.com/domonda/go-sqldb/mysqlconn => ../../mysqlconn
)

require (
	github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000
	github.com/domonda/go-sqldb/mysqlconn v0.0.0-00010101000000-000000000000
	github.com/go-sql-driver/mysql v1.9.2
)

require (
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/DataDog/go-sqllexer v0.1.13 // indirect
	github.com/corazawaf/libinjection-go v0.3.2 // indirect
)
