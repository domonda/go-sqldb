module github.com/domonda/go-sqldb/mysqlconn

go 1.23

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/go-sql-driver/mysql v1.9.2

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/DataDog/go-sqllexer v0.1.6 // indirect
	github.com/domonda/go-types v0.0.0-20250730131538-21e4dbd92676 // indirect
)
