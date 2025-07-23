module github.com/domonda/go-sqldb/cmd/sqldb-dump

go 1.24

toolchain go1.24.4

replace github.com/domonda/go-sqldb => ../..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/domonda/go-pretty v0.0.0-20250602142956-1b467adc6387
	github.com/domonda/go-sqldb/pqconn v0.0.0-20250721084848-23215f3b8988
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/domonda/go-errs v0.0.0-20250603150208-71d6de0c48ea // indirect
	github.com/domonda/go-types v0.0.0-20250711130302-a138ad20cd49 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
