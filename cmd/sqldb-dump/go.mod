module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.17

require (
	github.com/domonda/go-pretty v0.0.0-20210202131827-b4ff5dbd12fd
	github.com/domonda/go-sqldb v0.0.0-20210916082522-ed8ba45cac41
)

require github.com/lib/pq v1.10.3 // indirect

replace github.com/domonda/go-sqldb => ../..
