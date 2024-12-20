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

	"github.com/domonda/go-errs"
	"github.com/domonda/go-sqldb/db"
	"github.com/domonda/go-types/uu"
)

type PrimaryKeyColumn struct {
	Table      string `db:"table"`
	Column     string `db:"column"`
	Type       string `db:"type"`
	ForeignKey bool   `db:"foreign_key"`
}

func GetPrimaryKeyColumns(ctx context.Context) (cols []PrimaryKeyColumn, err error) {
	defer errs.WrapWithFuncParams(&err, ctx)

	err = db.QueryRows(ctx, `
		select
			tc.table_schema||'.'||tc.table_name as "table",
			kc.column_name                      as "column",
			col.data_type                       as "type",
			(select exists(
				select from information_schema.table_constraints as fk_tc
				inner join information_schema.key_column_usage as fk_kc
					on fk_kc.table_schema = fk_tc.table_schema
					and fk_kc.table_name = fk_tc.table_name
					and fk_kc.constraint_name = fk_tc.constraint_name
				where fk_tc.constraint_type = 'FOREIGN KEY'
					and fk_tc.table_schema = tc.table_schema
					and fk_tc.table_name = tc.table_name
					and fk_kc.column_name = kc.column_name
			)) as "foreign_key"
		from information_schema.table_constraints as tc
		inner join information_schema.key_column_usage as kc
			on kc.table_schema = tc.table_schema
			and kc.table_name = tc.table_name
			and kc.constraint_name = tc.constraint_name
		inner join information_schema.columns as col
			on col.table_schema = tc.table_schema
			and col.table_name = tc.table_name
			and col.column_name = kc.column_name
		where tc.constraint_type = 'PRIMARY KEY'
			and kc.ordinal_position is not null
		order by
			tc.table_schema,
			tc.table_name`,
	).ScanStructSlice(&cols)
	if err != nil {
		return nil, err
	}
	return cols, nil
}

func GetPrimaryKeyColumnsOfType(ctx context.Context, pkType string) (cols []PrimaryKeyColumn, err error) {
	defer errs.WrapWithFuncParams(&err, ctx, pkType)

	err = db.QueryRows(ctx, `
		select
			tc.table_schema||'.'||tc.table_name as "table",
			kc.column_name                      as "column",
			col.data_type                       as "type",
			(select exists(
				select from information_schema.table_constraints as fk_tc
				inner join information_schema.key_column_usage as fk_kc
					on fk_kc.table_schema = fk_tc.table_schema
					and fk_kc.table_name = fk_tc.table_name
					and fk_kc.constraint_name = fk_tc.constraint_name
				where fk_tc.constraint_type = 'FOREIGN KEY'
					and fk_tc.table_schema = tc.table_schema
					and fk_tc.table_name = tc.table_name
					and fk_kc.column_name = kc.column_name
			)) as "foreign_key"
		from information_schema.table_constraints as tc
		inner join information_schema.key_column_usage as kc
			on kc.table_schema = tc.table_schema
			and kc.table_name = tc.table_name
			and kc.constraint_name = tc.constraint_name
		inner join information_schema.columns as col
			on col.table_schema = tc.table_schema
			and col.table_name = tc.table_name
			and col.column_name = kc.column_name
		where tc.constraint_type = 'PRIMARY KEY'
			and kc.ordinal_position is not null
			and col.data_type = $1
		order by
			tc.table_schema,
			tc.table_name`,
		pkType,
	).ScanStructSlice(&cols)
	if err != nil {
		return nil, err
	}
	return cols, nil
}

type TableRowWithPrimaryKey struct {
	PrimaryKeyColumn
	Header []string
	Row    []string
}

func GetTableRowsWithPrimaryKey(ctx context.Context, pkCols []PrimaryKeyColumn, pk any) (tableRows []TableRowWithPrimaryKey, err error) {
	defer errs.WrapWithFuncParams(&err, ctx, pkCols, pk)

	for _, col := range pkCols {
		query := fmt.Sprintf(`SELECT * FROM %s WHERE "%s" = $1`, col.Table, col.Column)
		rows := db.QueryRows(ctx, query, pk)
		cols, err := rows.Columns()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		vals, err := rows.ScanAllRowsAsStrings(false)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		if len(vals) == 0 {
			continue
		}
		tableRows = append(tableRows, TableRowWithPrimaryKey{
			PrimaryKeyColumn: col,
			Header:           cols,
			Row:              vals[0],
		})
	}
	return tableRows, nil
}

var RenderUUIDPrimaryKeyRefsHTML = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
	var (
		title       string
		mainContent any
		style       = []string{StyleAllMonospace, StyleDefaultTable, `<style>h1 {color:red}</style>`}
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
		fmt.Fprintf(&b, "<h2><button onclick='navigator.clipboard.writeText(%q)'>Copy UUID</button></h2>", pk)
		for _, tableRow := range tableRows { //#nosec
			fmt.Fprintf(&b, "<h3>%s</h3>", html.EscapeString(tableRow.Table))
			fmt.Fprintf(&b, "<table>")
			for col, title := range tableRow.Header {
				val := tableRow.Row[col]
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
		*, *::before, *::after {
			box-sizing: border-box;
		}
		body {
			color: #21353e;
			font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
		}
		pre {
			background-color: #eee;
		}
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
	table > caption {
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
	td > button {
		font-size: 1em;
	}
	td > input[type="checkbox"] {
		width: 1.2em;
		height: 1.2em;
	}
}
</style>`
