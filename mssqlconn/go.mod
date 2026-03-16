module github.com/domonda/go-sqldb/mssqlconn

go 1.24.6

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/microsoft/go-mssqldb v1.9.2
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/DataDog/go-sqllexer v0.1.9 // indirect
	github.com/corazawaf/libinjection-go v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/domonda/go-types v0.0.0-20260220135934-fbc645f0b26b // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
