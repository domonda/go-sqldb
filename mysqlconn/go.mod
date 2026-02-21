module github.com/domonda/go-sqldb/mysqlconn

go 1.24.6

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/go-sql-driver/mysql v1.9.2
	github.com/stretchr/testify v1.11.1
)

require (
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/DataDog/go-sqllexer v0.1.9 // indirect
	github.com/corazawaf/libinjection-go v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/domonda/go-types v0.0.0-20260220135934-fbc645f0b26b // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
