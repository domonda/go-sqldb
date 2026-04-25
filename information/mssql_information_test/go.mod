module github.com/domonda/go-sqldb/information/mssql_information_test

go 1.24.6

replace (
	github.com/domonda/go-sqldb => ../..
	github.com/domonda/go-sqldb/mssqlconn => ../../mssqlconn
)

require (
	github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000
	github.com/domonda/go-sqldb/mssqlconn v0.0.0-00010101000000-000000000000
	github.com/microsoft/go-mssqldb v1.9.2
)

require (
	github.com/DataDog/go-sqllexer v0.1.13 // indirect
	github.com/corazawaf/libinjection-go v0.3.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
