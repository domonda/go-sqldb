module github.com/domonda/go-sqldb/pqconn

go 1.24.6

replace github.com/domonda/go-sqldb => ..

require github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/domonda/go-types v0.0.0-20260220135934-fbc645f0b26b
	github.com/lib/pq v1.11.2
)

require (
	github.com/DataDog/go-sqllexer v0.1.9 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/corazawaf/libinjection-go v0.2.3 // indirect
	github.com/domonda/go-pretty v1.0.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
