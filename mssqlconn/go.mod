module github.com/domonda/go-sqldb/mssqlconn

go 1.24

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/microsoft/go-mssqldb v1.8.0

require (
	github.com/DataDog/go-sqllexer v0.1.3 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)
