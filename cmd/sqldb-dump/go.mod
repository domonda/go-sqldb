module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.17

require (
	github.com/domonda/go-pretty v0.0.0-20220111114354-b0f333b12874
	github.com/domonda/go-sqldb v0.0.0-20211019065955-d4a19e17a722
)

require (
	github.com/lib/pq v1.10.4 // indirect
)

replace github.com/domonda/go-sqldb => ../..
