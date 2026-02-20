package sqldb

import "testing"

var testFormatter = NewQueryFormatter("$") // PostgreSQL style

func TestStdQueryBuilder_QueryRowWithPK(t *testing.T) {
	b := StdQueryBuilder{}

	tests := []struct {
		name      string
		table     string
		pkColumns []string
		want      string
	}{
		{
			name:      "single PK column",
			table:     "users",
			pkColumns: []string{"id"},
			want:      `SELECT * FROM users WHERE id = $1`,
		},
		{
			name:      "composite PK",
			table:     "order_items",
			pkColumns: []string{"order_id", "item_id"},
			want:      `SELECT * FROM order_items WHERE order_id = $1 AND item_id = $2`,
		},
		{
			name:      "three PK columns",
			table:     "tri_key",
			pkColumns: []string{"a", "b", "c"},
			want:      `SELECT * FROM tri_key WHERE a = $1 AND b = $2 AND c = $3`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.QueryRowWithPK(testFormatter, tt.table, tt.pkColumns)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}
}

func TestStdQueryBuilder_Insert(t *testing.T) {
	b := StdQueryBuilder{}

	tests := []struct {
		name    string
		table   string
		columns []ColumnInfo
		want    string
	}{
		{
			name:    "single column",
			table:   "users",
			columns: []ColumnInfo{{Name: "name"}},
			want:    `INSERT INTO users(name) VALUES($1)`,
		},
		{
			name:  "multiple columns",
			table: "users",
			columns: []ColumnInfo{
				{Name: "id"},
				{Name: "name"},
				{Name: "email"},
			},
			want: `INSERT INTO users(id,name,email) VALUES($1,$2,$3)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.Insert(testFormatter, tt.table, tt.columns)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}
}

func TestStdQueryBuilder_InsertUnique(t *testing.T) {
	b := StdQueryBuilder{}

	tests := []struct {
		name       string
		table      string
		columns    []ColumnInfo
		onConflict string
		want       string
	}{
		{
			name:       "with parenthesized onConflict",
			table:      "users",
			columns:    []ColumnInfo{{Name: "id"}, {Name: "name"}},
			onConflict: "(id,name)",
			want:       `INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id,name) DO NOTHING RETURNING TRUE`,
		},
		{
			name:       "without parens",
			table:      "users",
			columns:    []ColumnInfo{{Name: "id"}, {Name: "name"}},
			onConflict: "id,name",
			want:       `INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id,name) DO NOTHING RETURNING TRUE`,
		},
		{
			name:       "single conflict column",
			table:      "users",
			columns:    []ColumnInfo{{Name: "id"}, {Name: "email"}},
			onConflict: "id",
			want:       `INSERT INTO users(id,email) VALUES($1,$2) ON CONFLICT (id) DO NOTHING RETURNING TRUE`,
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

func TestStdQueryBuilder_Upsert(t *testing.T) {
	b := StdQueryBuilder{}

	tests := []struct {
		name    string
		table   string
		columns []ColumnInfo
		want    string
	}{
		{
			name:  "single PK and single value",
			table: "users",
			columns: []ColumnInfo{
				{Name: "id", PrimaryKey: true},
				{Name: "name"},
			},
			want: `INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET name=$2`,
		},
		{
			name:  "composite PK and multiple values",
			table: "order_items",
			columns: []ColumnInfo{
				{Name: "order_id", PrimaryKey: true},
				{Name: "item_id", PrimaryKey: true},
				{Name: "quantity"},
				{Name: "price"},
			},
			want: `INSERT INTO order_items(order_id,item_id,quantity,price) VALUES($1,$2,$3,$4) ON CONFLICT(order_id,item_id) DO UPDATE SET quantity=$3, price=$4`,
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
}

func TestStdQueryBuilder_Update(t *testing.T) {
	b := StdQueryBuilder{}

	tests := []struct {
		name         string
		table        string
		values       Values
		where        string
		whereArgs    []any
		wantQuery    string
		wantArgCount int
	}{
		{
			name:         "single value no whereArgs",
			table:        "users",
			values:       Values{"name": "Alice"},
			where:        "id = 42",
			whereArgs:    nil,
			wantQuery:    `UPDATE users SET name=$1 WHERE id = 42`,
			wantArgCount: 1,
		},
		{
			name:         "multiple values with whereArgs",
			table:        "users",
			values:       Values{"email": "a@b.com", "name": "Bob"},
			where:        "id = $1 AND active = $2",
			whereArgs:    []any{1, true},
			wantQuery:    `UPDATE users SET email=$3, name=$4 WHERE id = $1 AND active = $2`,
			wantArgCount: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotArgs, err := b.Update(testFormatter, tt.table, tt.values, tt.where, tt.whereArgs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query:\n  got:  %s\n  want: %s", gotQuery, tt.wantQuery)
			}
			if len(gotArgs) != tt.wantArgCount {
				t.Errorf("args count: got %d, want %d", len(gotArgs), tt.wantArgCount)
			}
		})
	}
}

func TestStdQueryBuilder_UpdateColumns(t *testing.T) {
	b := StdQueryBuilder{}

	tests := []struct {
		name    string
		table   string
		columns []ColumnInfo
		want    string
	}{
		{
			name:  "single PK and single value",
			table: "users",
			columns: []ColumnInfo{
				{Name: "id", PrimaryKey: true},
				{Name: "name"},
			},
			want: `UPDATE users SET name=$2 WHERE id = $1`,
		},
		{
			name:  "composite PK and multiple values",
			table: "order_items",
			columns: []ColumnInfo{
				{Name: "order_id", PrimaryKey: true},
				{Name: "item_id", PrimaryKey: true},
				{Name: "quantity"},
				{Name: "price"},
			},
			want: `UPDATE order_items SET quantity=$3, price=$4 WHERE order_id = $1 AND item_id = $2`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.UpdateColumns(testFormatter, tt.table, tt.columns)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}
}
