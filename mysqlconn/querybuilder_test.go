package mysqlconn

import (
	"testing"

	"github.com/domonda/go-sqldb"
)

var testFormatter = QueryFormatter{} // MySQL ? placeholders

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
			want: "INSERT INTO users(id,name) VALUES(?,?) ON DUPLICATE KEY UPDATE name=VALUES(name)",
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
			want: "INSERT INTO order_items(order_id,item_id,quantity,price) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE quantity=VALUES(quantity), price=VALUES(price)",
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

func TestQueryBuilder_InsertUnique_error(t *testing.T) {
	b := QueryBuilder{}
	_, err := b.InsertUnique(testFormatter, "users", []sqldb.ColumnInfo{
		{Name: "id"},
		{Name: "name"},
	}, "id")
	if err == nil {
		t.Error("expected error from MySQL InsertUnique")
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
