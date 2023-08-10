module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.21

require (
	github.com/domonda/go-pretty v0.0.0-20230810130018-8920f571470a
	github.com/domonda/go-sqldb v0.0.0-20220406183832-9d70b61cac63
)

require (
	github.com/domonda/go-errs v0.0.0-20230810132956-1b6272f9fc8f // indirect
	github.com/domonda/go-types v0.0.0-20230810134814-bd15ee23faf5 // indirect
	github.com/lib/pq v1.10.9 // indirect
)

replace github.com/domonda/go-sqldb => ../..
