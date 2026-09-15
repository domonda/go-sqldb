module github.com/domonda/go-sqldb/mysqlconn

go 1.26.0

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/go-sql-driver/mysql v1.10.1
	github.com/stretchr/testify v1.12.1
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/DataDog/go-sqllexer v0.2.4 // indirect
	github.com/corazawaf/libinjection-go v0.3.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
