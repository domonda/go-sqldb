module github.com/domonda/go-sqldb/mssqlconn

go 1.23

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/microsoft/go-mssqldb v1.9.2

require (
	github.com/DataDog/go-sqllexer v0.1.6 // indirect
	github.com/domonda/go-types v0.0.0-20250730131538-21e4dbd92676 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)
