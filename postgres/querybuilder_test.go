package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

var testFormatter = sqldb.NewQueryFormatter("$")

func TestQueryBuilder_InsertUnique(t *testing.T) {
	b := QueryBuilder{}

	for _, scenario := range []struct {
		name       string
		table      string
		columns    []sqldb.ColumnInfo
		onConflict string
		wantQuery  string
	}{
		{
			name:  "single column PK with parenthesized onConflict",
			table: "users",
			columns: []sqldb.ColumnInfo{
				{Name: "id", PrimaryKey: true},
				{Name: "name"},
				{Name: "email"},
			},
			onConflict: "(id)",
			wantQuery:  `INSERT INTO users(id,name,email) VALUES($1,$2,$3) ON CONFLICT (id) DO NOTHING`,
		},
		{
			name:  "two column onConflict without parens",
			table: "order_items",
			columns: []sqldb.ColumnInfo{
				{Name: "order_id", PrimaryKey: true},
				{Name: "item_id", PrimaryKey: true},
				{Name: "quantity"},
			},
			onConflict: "order_id,item_id",
			wantQuery:  `INSERT INTO order_items(order_id,item_id,quantity) VALUES($1,$2,$3) ON CONFLICT (order_id,item_id) DO NOTHING`,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			formatter := testFormatter
			columns := scenario.columns

			// when
			query, err := b.InsertUnique(formatter, scenario.table, columns, scenario.onConflict)

			// then
			require.NoError(t, err)
			assert.Equal(t, scenario.wantQuery, query)
			assert.Contains(t, query, "ON CONFLICT")
			assert.Contains(t, query, "DO NOTHING")
		})
	}
}

func TestQueryBuilder_Upsert(t *testing.T) {
	b := QueryBuilder{}

	for _, scenario := range []struct {
		name      string
		table     string
		columns   []sqldb.ColumnInfo
		wantQuery string
		wantErr   string
	}{
		{
			name:  "single PK and single value column",
			table: "users",
			columns: []sqldb.ColumnInfo{
				{Name: "id", PrimaryKey: true},
				{Name: "name"},
			},
			wantQuery: `INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id) DO UPDATE SET name=$2`,
		},
		{
			name:  "single PK and multiple value columns",
			table: "users",
			columns: []sqldb.ColumnInfo{
				{Name: "id", PrimaryKey: true},
				{Name: "name"},
				{Name: "email"},
				{Name: "age"},
			},
			wantQuery: `INSERT INTO users(id,name,email,age) VALUES($1,$2,$3,$4) ON CONFLICT (id) DO UPDATE SET name=$2, email=$3, age=$4`,
		},
		{
			name:  "multiple PK columns and value column",
			table: "order_items",
			columns: []sqldb.ColumnInfo{
				{Name: "order_id", PrimaryKey: true},
				{Name: "item_id", PrimaryKey: true},
				{Name: "quantity"},
			},
			wantQuery: `INSERT INTO order_items(order_id,item_id,quantity) VALUES($1,$2,$3) ON CONFLICT (order_id,item_id) DO UPDATE SET quantity=$3`,
		},
		{
			name:  "all columns are PK returns error",
			table: "tags",
			columns: []sqldb.ColumnInfo{
				{Name: "id", PrimaryKey: true},
				{Name: "name", PrimaryKey: true},
			},
			wantErr: "Upsert requires at least one non-primary-key column",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			formatter := testFormatter
			columns := scenario.columns

			// when
			query, err := b.Upsert(formatter, scenario.table, columns)

			// then
			if scenario.wantErr != "" {
				require.EqualError(t, err, scenario.wantErr)
				assert.Empty(t, query)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, scenario.wantQuery, query)
			assert.Contains(t, query, "ON CONFLICT")
			assert.Contains(t, query, "DO UPDATE SET")
		})
	}
}
