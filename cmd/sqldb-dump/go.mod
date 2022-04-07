module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.18

require (
	github.com/domonda/go-pretty v0.0.0-20220317123925-dd9e6bef129a
	github.com/domonda/go-sqldb v0.0.0-20220406183832-9d70b61cac63
)

require (
	github.com/domonda/go-types v0.0.0-20220406183531-66c6125b4053 // indirect
	github.com/lib/pq v1.10.4 // indirect
)

replace github.com/domonda/go-sqldb => ../..
