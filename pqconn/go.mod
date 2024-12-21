module github.com/domonda/go-sqldb/pqconn

go 1.23

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/domonda/go-types v0.0.0-20241220151539-e2cc4555efcb
	github.com/lib/pq v1.10.9
)

require (
	github.com/DataDog/go-sqllexer v0.0.18 // indirect
	github.com/domonda/go-pretty v0.0.0-20240110134850-17385799142f // indirect
)
