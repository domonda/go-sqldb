module github.com/domonda/go-sqldb/pqconn

go 1.24

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/domonda/go-types v0.0.0-20250327120403-f5ca1ab99ab1
	github.com/lib/pq v1.10.9
)

require (
	github.com/DataDog/go-sqllexer v0.1.3 // indirect
	github.com/domonda/go-pretty v0.0.0-20240110134850-17385799142f // indirect
)
