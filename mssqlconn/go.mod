module github.com/domonda/go-sqldb/mssqlconn

go 1.26.0

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/microsoft/go-mssqldb v1.11.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/DataDog/go-sqllexer v0.2.4 // indirect
	github.com/corazawaf/libinjection-go v0.3.3 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
