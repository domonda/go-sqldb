module github.com/domonda/go-sqldb/mysqlconn

go 1.22.0

replace github.com/domonda/go-sqldb => ../

require (
	github.com/domonda/go-sqldb v0.0.0-20240122205319-56db59ae89d2
	github.com/go-sql-driver/mysql v1.7.1
)

require (
	github.com/domonda/go-types v0.0.0-20240301143218-7f4371e713b4 // indirect
	github.com/lib/pq v1.10.9 // indirect
)
