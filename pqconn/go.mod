module github.com/domonda/go-sqldb/pqconn

go 1.26.0

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/lib/pq v1.12.3
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/DataDog/go-sqllexer v0.2.4 // indirect
	github.com/corazawaf/libinjection-go v0.3.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
