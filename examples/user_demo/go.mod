module github.com/domonda/go-sqldb/examples/user_demo

go 1.24

replace (
	github.com/domonda/go-sqldb => ../..
	github.com/domonda/go-sqldb/pqconn => ../../pqconn
)

require (
	github.com/domonda/go-sqldb v0.0.0-00010101000000-000000000000 // replaced
	github.com/domonda/go-sqldb/pqconn v0.0.0-00010101000000-000000000000 // replaced
)

require github.com/domonda/go-types v0.0.0-20250725104804-d473d8b9dd16

require (
	github.com/DataDog/go-sqllexer v0.1.6 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cention-sany/utf7 v0.0.0-20170124080048-26cad61bd60a // indirect
	github.com/domonda/go-errs v0.0.0-20250603150208-71d6de0c48ea // indirect
	github.com/domonda/go-pretty v0.0.0-20250602142956-1b467adc6387 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/jaytaylor/html2text v0.0.0-20230321000545-74c2419ad056 // indirect
	github.com/jhillyerd/enmime v1.3.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/teamwork/tnef v0.0.0-20200108124832-7deabccfdb32 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/xurls/v2 v2.6.0 // indirect
)
