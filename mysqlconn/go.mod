module github.com/domonda/go-sqldb/mysqlconn

go 1.23

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require github.com/go-sql-driver/mysql v1.7.1

require (
	github.com/domonda/go-types v0.0.0-20240822142828-3b45a403e1e2 // indirect
	github.com/lib/pq v1.10.9 // indirect
)
