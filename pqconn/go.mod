module github.com/domonda/go-sqldb/pqconn

go 1.24.0

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/domonda/go-types v0.0.0-20251013130956-d9f0fe9a7e07
	github.com/lib/pq v1.10.9
)

require (
	github.com/DataDog/go-sqllexer v0.1.8 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/corazawaf/libinjection-go v0.2.2 // indirect
	github.com/domonda/go-pretty v0.0.0-20250602142956-1b467adc6387 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
