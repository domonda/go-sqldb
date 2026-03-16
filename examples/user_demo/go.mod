module github.com/domonda/go-sqldb/examples/user_demo

go 1.24.6

replace (
	github.com/domonda/go-sqldb => ../..
	github.com/domonda/go-sqldb/pqconn => ../../pqconn
)

require (
	github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced
	github.com/domonda/go-sqldb/pqconn v0.0.0-00010101000000-000000000000 // replaced
)

require github.com/domonda/go-types v0.0.0-20260220135934-fbc645f0b26b

require (
	github.com/DataDog/go-sqllexer v0.1.9 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cention-sany/utf7 v0.0.0-20170124080048-26cad61bd60a // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.10.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.6.0 // indirect
	github.com/corazawaf/libinjection-go v0.2.3 // indirect
	github.com/domonda/go-errs v1.0.0 // indirect
	github.com/domonda/go-pretty v1.0.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/inbucket/html2text v1.0.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/jhillyerd/enmime/v2 v2.3.0 // indirect
	github.com/lib/pq v1.11.2 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.2.0 // indirect
	github.com/olekukonko/ll v0.1.6 // indirect
	github.com/olekukonko/tablewriter v1.1.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/teamwork/tnef v0.0.0-20200108124832-7deabccfdb32 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/xurls/v2 v2.6.0 // indirect
)
