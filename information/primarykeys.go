package information

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"
	"text/template"

	"github.com/domonda/go-sqldb/db"
	"github.com/domonda/go-types/uu"
)

// PrimaryKeyColumn holds information about a primary key column
// including whether it is also a foreign key.
type PrimaryKeyColumn struct {
	Table      string `db:"table"`
	Column     string `db:"column"`
	Type       string `db:"type"`
	ForeignKey bool   `db:"foreign_key"`
}

func GetPrimaryKeyColumns(ctx context.Context) (cols []PrimaryKeyColumn, err error) {
	return db.QueryRowsAsSlice[PrimaryKeyColumn](ctx,
		/*sql*/ `
		SELECT
			tc.table_schema||'.'||tc.table_name AS "table",
			kc.column_name                      AS "column",
			col.data_type                       AS "type",
			(SELECT EXISTS(
				SELECT FROM information_schema.table_constraints AS fk_tc
				JOIN information_schema.key_column_usage AS fk_kc
					ON fk_kc.table_schema = fk_tc.table_schema
					AND fk_kc.table_name = fk_tc.table_name
					AND fk_kc.constraint_name = fk_tc.constraint_name
				WHERE fk_tc.constraint_type = 'FOREIGN KEY'
					AND fk_tc.table_schema = tc.table_schema
					AND fk_tc.table_name = tc.table_name
					AND fk_kc.column_name = kc.column_name
			)) AS "foreign_key"
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kc
			ON kc.table_schema = tc.table_schema
			AND kc.table_name = tc.table_name
			AND kc.constraint_name = tc.constraint_name
		JOIN information_schema.columns AS col
			ON col.table_schema = tc.table_schema
			AND col.table_name = tc.table_name
			AND col.column_name = kc.column_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND kc.ordinal_positiON IS NOT NULL
		ORDER BY
			tc.table_schema,
			tc.table_name`,
	)
}

func GetPrimaryKeyColumnsOfType(ctx context.Context, pkType string) (cols []PrimaryKeyColumn, err error) {
	return db.QueryRowsAsSlice[PrimaryKeyColumn](ctx,
		/*sql*/ `
		SELECT
			tc.table_schema||'.'||tc.table_name AS "table",
			kc.column_name                      AS "column",
			col.data_type                       AS "type",
			(SELECT EXISTS(
				SELECT FROM information_schema.table_constraints AS fk_tc
				JOIN information_schema.key_column_usage AS fk_kc
					ON fk_kc.table_schema = fk_tc.table_schema
					AND fk_kc.table_name = fk_tc.table_name
					AND fk_kc.constraint_name = fk_tc.constraint_name
				WHERE fk_tc.constraint_type = 'FOREIGN KEY'
					AND fk_tc.table_schema = tc.table_schema
					AND fk_tc.table_name = tc.table_name
					AND fk_kc.column_name = kc.column_name
			)) AS "foreign_key"
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kc
			ON kc.table_schema = tc.table_schema
			AND kc.table_name = tc.table_name
			AND kc.constraint_name = tc.constraint_name
		JOIN information_schema.columns AS col
			ON col.table_schema = tc.table_schema
			AND col.table_name = tc.table_name
			AND col.column_name = kc.column_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND kc.ordinal_positiON IS NOT NULL
			AND col.data_type = $1
		ORDER BY
			tc.table_schema,
			tc.table_name`,
		pkType,
	)
}

type TableRowWithPrimaryKey struct {
	PrimaryKeyColumn
	Header []string
	Row    []string
}

func GetTableRowsWithPrimaryKey(ctx context.Context, pkCols []PrimaryKeyColumn, pk any) (tableRows []TableRowWithPrimaryKey, err error) {
	for _, col := range pkCols {
		query := fmt.Sprintf(`SELECT * FROM %s WHERE "%s" = $1`, col.Table, col.Column)
		strs, err := db.QueryRowsAsStrings(ctx, query, pk)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		if len(strs) < 2 {
			continue
		}
		tableRows = append(tableRows, TableRowWithPrimaryKey{
			PrimaryKeyColumn: col,
			Header:           strs[0],
			Row:              strs[1],
		})
	}
	return tableRows, nil
}

var RenderUUIDPrimaryKeyRefsHTML = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
	var (
		title       string
		mainContent any
		style       = []string{
			StyleAllMonospace,
			StyleDefaultTable,
			`<style>h1 {color:red}</style>`,
		}
	)
	pk, err := uu.IDFromString(request.URL.Query().Get("pk"))
	if err != nil {
		title = "Primary Key UUID"
		mainContent = /*html*/ `
			<form onsubmit="event.preventDefault();location='.?pk='+encodeURIComponent(document.getElementById('uuid').value.trim())">
				<input type="text" size="40" id="uuid"/>
				<input type="submit" value="Look up"/>
			</form>`
	} else {
		title = pk.String()
		ctx := request.Context()
		cols, err := GetPrimaryKeyColumnsOfType(ctx, "uuid")
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		tableRows, err := GetTableRowsWithPrimaryKey(ctx, cols, pk)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		sort.SliceStable(tableRows, func(i, j int) bool {
			return !tableRows[i].ForeignKey && tableRows[j].ForeignKey
		})
		var b strings.Builder
		fmt.Fprintf(&b, "<h2><buttON onclick='navigator.clipboard.writeText(%q)'>Copy UUID</button></h2>", pk)
		for _, tableRow := range tableRows { //#nosec
			fmt.Fprintf(&b, "<h3>%s</h3>", html.EscapeString(tableRow.Table))
			fmt.Fprintf(&b, "<table>")
			for colIdx, title := range tableRow.Header {
				val := tableRow.Row[colIdx]
				id, err := uu.IDFromString(val)
				if err == nil {
					if id == pk {
						var fk string
						if tableRow.ForeignKey {
							fk = " (foreign key)"
						}
						fmt.Fprintf(&b, "<tr><td>%s</td><td><b style='color:red'>%s</b>%s</td></tr>", html.EscapeString(title), id, fk)
					} else {
						fmt.Fprintf(&b, "<tr><td>%[1]s</td><td><a href='.?pk=%[2]s'>%[2]s</a></td></tr>", html.EscapeString(title), id)
					}
				} else {
					fmt.Fprintf(&b, "<tr><td>%s</td><td>%s</td></tr>", html.EscapeString(title), html.EscapeString(val))
				}
			}
			fmt.Fprintf(&b, "</table>")
		}
		mainContent = b.String()
	}

	tpl, err := template.New("").Parse(htmlTemplate)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, map[string]any{
		"title":       title,
		"style":       style,
		"headerTitle": true,
		"mainContent": mainContent,
	})
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	writer.Write(buf.Bytes()) //#nosec G104
})

var htmlTemplate = /*html*/ `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>{{html .title}}</title>
	<style>
		:root {
			--bg: #ffffff;
			--text: #222222;
			--link: #0a7ea4;
			--link-visited: #6a4fb3;
			--space: 1rem;
		}
		*, *::before, *::after { box-sizing: border-box; }
		html { 
			margin: 8px;
			padding: 0;
			color-scheme: light;
		}
		body {
			margin: 0;
			padding: 0;
			background: var(--bg);
			color: var(--text);
			font-family: system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif;
			line-height: 1.6;
			font-size: 16px;
			-webkit-font-smoothing: antialiased;
			text-rendering: optimizeLegibility;
		}
		label { display: block; }
		form { margin: 10px; }
		form div { padding-bottom: 10px; }
		p, ul, ol, blockquote, pre { margin: 0 0 var(--space); }
		ul, ol { padding-left: 1.25rem; }
		h1, h2, h3, h4, h5, h6 {
			line-height: 1.2;
			margin: 1.5rem 0 0.5rem;
		}
		a {
			color: var(--link);
			text-underline-offset: 0.15em;
		}
		a:visited { color: var(--link-visited); }
		a:hover, a:focus { text-decoration: underline; }
		:focus-visible {
			outline: 3px solid #ffbf47; /* high-contrast focus */
			outline-offset: 2px;
		}
		/* Code blocks */
		pre { background-color: #eee; }
		.monospace {
			font-family: "Lucida Console", Monaco, monospace;
		}
	</style>{{range .style}}
	{{.}}{{end}}
</head>
<body>
	{{if .headerTitle}}<header>
		<h1>{{html .title}}</h1>
	</header>{{end}}
	<main>
		{{printf "%s" .mainContent}}
	</main>
</body>
</html>`

const StyleAllMonospace = /*html*/ `<style>* { font-family: "Lucida Console", Monaco, monospace; }</style>`

const StyleDefaultTable = /*html*/ `<style>
	table {
		margin-top: 1em;
		margin-bottom: 1em;
	}
	table, td, th {
		border-collapse: collapse;
		border: 1px solid black;
		padding: 4px;
		white-space: nowrap;
		font-family: "Lucida Console", Monaco, monospace;
	}
	table > captiON {
		font-size: 1.4em;
		text-align: left;
		margin-bottom: 8px;
		font-family: Arial, sans-serif;
	}
	tr:nth-child(odd) > th, tr:nth-child(odd) > td {
		background: #EEF;
	}
	tr:nth-child(even) > th, tr:nth-child(even) > td {
		background: #FFF;
	}
	th {
		position: sticky;
		top: 0;
		z-index: 1;
	}
	th:first-child, td:first-child {
		position: sticky;
		left: 0;
		z-index: 2;
	}
	th:first-child {
		position: sticky;
		top: 0;
		left: 0;
		z-index: 3;
	}
	td > pre, td > code {
		margin: 0;
		font-family: "Lucida Console", Monaco, monospace;
	}
	td > buttON {
		font-size: 1em;
	}
	td > input[type="checkbox"] {
		width: 1.2em;
		height: 1.2em;
	}
}
</style>`
