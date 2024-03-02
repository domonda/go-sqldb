module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.22

replace github.com/domonda/go-sqldb => ../..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/domonda/go-pretty v0.0.0-20240110134850-17385799142f

require (
	github.com/domonda/go-errs v0.0.0-20240301142737-8fde935c9bd4 // indirect
	github.com/domonda/go-types v0.0.0-20240301143218-7f4371e713b4 // indirect
	github.com/lib/pq v1.10.9 // indirect
)
