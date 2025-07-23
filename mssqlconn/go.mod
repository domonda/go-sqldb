module github.com/domonda/go-sqldb/mssqlconn

go 1.23

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/microsoft/go-mssqldb v1.9.2

require (
	github.com/domonda/go-types v0.0.0-20250711130302-a138ad20cd49 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)
