module github.com/domonda/go-sqldb/mysqlconn

go 1.23

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/go-sql-driver/mysql v1.9.0

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/domonda/go-types v0.0.0-20250711130302-a138ad20cd49 // indirect
	github.com/lib/pq v1.10.9 // indirect
)
