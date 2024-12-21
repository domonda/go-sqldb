module github.com/domonda/go-sqldb/mssqlconn

go 1.23

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/microsoft/go-mssqldb v1.7.0

require (
	github.com/domonda/go-types v0.0.0-20241104173616-e85c6dede426 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)
