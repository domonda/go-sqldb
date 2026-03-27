module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.24.6

replace (
	github.com/domonda/go-sqldb => ../..
	github.com/domonda/go-sqldb/pqconn => ../../pqconn
)

require (
	github.com/domonda/go-pretty v1.0.0
	github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced
	github.com/domonda/go-sqldb/pqconn v0.0.0-00010101000000-000000000000 // replaced
)

require (
	github.com/DataDog/go-sqllexer v0.1.13 // indirect
	github.com/corazawaf/libinjection-go v0.3.2 // indirect
	github.com/lib/pq v1.12.0 // indirect
)
