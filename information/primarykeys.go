package information

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"text/template"

	"github.com/domonda/go-errs"
	"github.com/domonda/go-sqldb/db"
	"github.com/domonda/go-types/uu"
)

type PrimaryKeyColumn struct {
	Table  string `db:"table"`
	Column string `db:"column"`
	Type   string `db:"type"`
}

func GetPrimaryKeyColumns(ctx context.Context) (cols []PrimaryKeyColumn, err error) {
	defer errs.WrapWithFuncParams(&err, ctx)

	err = db.Conn(ctx).QueryRows(`
		select
			tc.table_schema||'.'||tc.table_name as "table",
			kc.column_name                      as "column",
			col.data_type                       as "type"
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
			tc.table_name,
			kc.position_in_unique_constraint`,
	).ScanStructSlice(&cols)
	if err != nil {
		return nil, err
	}
	return cols, nil
}

func GetPrimaryKeyColumnsOfType(ctx context.Context, pkType string) (cols []PrimaryKeyColumn, err error) {
	defer errs.WrapWithFuncParams(&err, ctx, pkType)

	err = db.Conn(ctx).QueryRows(`
		select
			tc.table_schema||'.'||tc.table_name as "table",
			kc.column_name                      as "column",
			col.data_type                       as "type"
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
			tc.table_name,
			kc.position_in_unique_constraint`,
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

	conn := db.Conn(ctx)
	for _, col := range pkCols {
		query := fmt.Sprintf(`select * from %s where "%s" = $1`, col.Table, col.Column)
		row := conn.QueryRows(query, pk)
		cols, err := row.Columns()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		vals, err := row.ScanAllRowsAsStrings(false)
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
		style       = []string{StyleAllMonospace, StyleDefaultTable}
	)
	pk, err := uu.IDFromString(request.URL.Query().Get("pk"))
	if err != nil {
		title = "Primary Key UUID"
		mainContent = `
			<form onsubmit="event.preventDefault();location='.?pk='+encodeURIComponent(document.getElementById('uuid').value.trim())">
				<input type="text" size="40" id="uuid"/>
				<input type="submit" value="Look up"/>
			</form>`
	} else {
		title = fmt.Sprintf("UUID %s", pk)
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
		var b strings.Builder
		for _, tableRow := range tableRows { //#nosec
			fmt.Fprintf(&b, "<h3>%s</h3>", html.EscapeString(tableRow.Table))
			fmt.Fprintf(&b, "<table>")
			for col, title := range tableRow.Header {
				val := tableRow.Row[col]
				id, err := uu.IDFromString(val)
				if err == nil {
					if id == pk {
						fmt.Fprintf(&b, "<tr><td>%s</td><td><b>%s</b></td></tr>", html.EscapeString(title), id)
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

var htmlTemplate = `<!DOCTYPE html>
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

const StyleAllMonospace = `<style>* { font-family: "Lucida Console", Monaco, monospace; }</style>`

const StyleDefaultTable = `<style>
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
