module github.com/domonda/go-sqldb/information/postgres_information_test

go 1.24.6

replace (
	github.com/domonda/go-sqldb => ../..
	github.com/domonda/go-sqldb/pqconn => ../../pqconn
)

require (
	github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000
	github.com/domonda/go-sqldb/pqconn v0.0.0-00010101000000-000000000000
)

require (
	github.com/DataDog/go-sqllexer v0.1.13 // indirect
	github.com/corazawaf/libinjection-go v0.3.2 // indirect
	github.com/lib/pq v1.12.3 // indirect
)
