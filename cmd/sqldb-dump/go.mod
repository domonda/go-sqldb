module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.17

require (
	github.com/domonda/go-pretty v0.0.0-20220203122647-ad6e3262a14c
	github.com/domonda/go-sqldb v0.0.0-20220228144720-61c5e0262412
)

require (
	github.com/domonda/go-types v0.0.0-20220228144452-dde8d262ead4 // indirect
	github.com/lib/pq v1.10.4 // indirect
)

replace github.com/domonda/go-sqldb => ../..
