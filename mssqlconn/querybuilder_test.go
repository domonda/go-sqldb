package mssqlconn

import (
	"testing"

	"github.com/domonda/go-sqldb"
)

var testFormatter = QueryFormatter{} // MSSQL @pN placeholders

func TestQueryBuilder_Upsert(t *testing.T) {
	b := QueryBuilder{}

	tests := []struct {
		name    string
		table   string
		columns []sqldb.ColumnInfo
		want    string
	}{
		{
			name:  "single PK and single value",
			table: "users",
			columns: []sqldb.ColumnInfo{
				{Name: "id", PrimaryKey: true},
				{Name: "name"},
			},
			want: `MERGE INTO users WITH (HOLDLOCK) AS target` +
				` USING (VALUES(@p1,@p2)) AS source(id,name)` +
				` ON target.id = source.id` +
				` WHEN MATCHED THEN UPDATE SET target.name = source.name` +
				` WHEN NOT MATCHED THEN INSERT (id,name) VALUES (source.id,source.name);`,
		},
		{
			name:  "composite PK and multiple values",
			table: "order_items",
			columns: []sqldb.ColumnInfo{
				{Name: "order_id", PrimaryKey: true},
				{Name: "item_id", PrimaryKey: true},
				{Name: "quantity"},
				{Name: "price"},
			},
			want: `MERGE INTO order_items WITH (HOLDLOCK) AS target` +
				` USING (VALUES(@p1,@p2,@p3,@p4)) AS source(order_id,item_id,quantity,price)` +
				` ON target.order_id = source.order_id AND target.item_id = source.item_id` +
				` WHEN MATCHED THEN UPDATE SET target.quantity = source.quantity, target.price = source.price` +
				` WHEN NOT MATCHED THEN INSERT (order_id,item_id,quantity,price) VALUES (source.order_id,source.item_id,source.quantity,source.price);`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.Upsert(testFormatter, tt.table, tt.columns)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}

	t.Run("all PK columns error", func(t *testing.T) {
		_, err := b.Upsert(testFormatter, "t", []sqldb.ColumnInfo{
			{Name: "id", PrimaryKey: true},
		})
		if err == nil {
			t.Error("expected error for all-PK columns")
		}
	})
}

func TestQueryBuilder_InsertUnique(t *testing.T) {
	b := QueryBuilder{}

	tests := []struct {
		name       string
		table      string
		columns    []sqldb.ColumnInfo
		onConflict string
		want       string
	}{
		{
			name:       "single conflict column",
			table:      "users",
			columns:    []sqldb.ColumnInfo{{Name: "id"}, {Name: "name"}},
			onConflict: "id",
			want: `MERGE INTO users WITH (HOLDLOCK) AS target` +
				` USING (VALUES(@p1,@p2)) AS source(id,name)` +
				` ON target.id = source.id` +
				` WHEN NOT MATCHED THEN INSERT (id,name) VALUES (source.id,source.name);`,
		},
		{
			name:       "multiple conflict columns",
			table:      "kv",
			columns:    []sqldb.ColumnInfo{{Name: "ns"}, {Name: "key"}, {Name: "val"}},
			onConflict: "ns, key",
			want: `MERGE INTO kv WITH (HOLDLOCK) AS target` +
				` USING (VALUES(@p1,@p2,@p3)) AS source(ns,[key],val)` +
				` ON target.ns = source.ns AND target.[key] = source.[key]` +
				` WHEN NOT MATCHED THEN INSERT (ns,[key],val) VALUES (source.ns,source.[key],source.val);`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.InsertUnique(testFormatter, tt.table, tt.columns, tt.onConflict)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}
}

func TestQueryBuilder_implements_interfaces(t *testing.T) {
	var b any = QueryBuilder{}
	if _, ok := b.(sqldb.QueryBuilder); !ok {
		t.Error("QueryBuilder should implement sqldb.QueryBuilder")
	}
	if _, ok := b.(sqldb.UpsertQueryBuilder); !ok {
		t.Error("QueryBuilder should implement sqldb.UpsertQueryBuilder")
	}
	if _, ok := b.(sqldb.ReturningQueryBuilder); ok {
		t.Error("QueryBuilder should NOT implement sqldb.ReturningQueryBuilder")
	}
}
