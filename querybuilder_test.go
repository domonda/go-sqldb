package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestStdQueryBuilder_DeleteColumns(t *testing.T) {
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
			columns: []ColumnInfo{{Name: "id", PrimaryKey: true}},
			want:    `DELETE FROM users WHERE id = $1`,
		},
		{
			name:  "composite PK",
			table: "order_items",
			columns: []ColumnInfo{
				{Name: "order_id", PrimaryKey: true},
				{Name: "item_id", PrimaryKey: true},
			},
			want: `DELETE FROM order_items WHERE order_id = $1 AND item_id = $2`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.Delete(testFormatter, tt.table, tt.columns)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n  %s\nwant:\n  %s", got, tt.want)
			}
		})
	}

	t.Run("no columns error", func(t *testing.T) {
		_, err := b.Delete(testFormatter, "users", nil)
		if err == nil {
			t.Error("expected error for empty columns")
		}
	})
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

	t.Run("all primary keys error", func(t *testing.T) {
		_, err := b.UpdateColumns(testFormatter, "keys_only", []ColumnInfo{
			{Name: "a", PrimaryKey: true},
			{Name: "b", PrimaryKey: true},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-primary-key column")
	})
}

func TestStdQueryBuilder_InsertRows(t *testing.T) {
	b := StdQueryBuilder{}

	t.Run("single row", func(t *testing.T) {
		got, err := b.InsertRows(testFormatter, "users", []ColumnInfo{
			{Name: "id"},
			{Name: "name"},
		}, 1)
		require.NoError(t, err)
		assert.Equal(t, `INSERT INTO users(id,name) VALUES($1,$2)`, got)
	})

	t.Run("multiple rows", func(t *testing.T) {
		got, err := b.InsertRows(testFormatter, "users", []ColumnInfo{
			{Name: "id"},
			{Name: "name"},
			{Name: "email"},
		}, 3)
		require.NoError(t, err)
		assert.Equal(t,
			`INSERT INTO users(id,name,email) VALUES($1,$2,$3),($4,$5,$6),($7,$8,$9)`,
			got,
		)
	})

	t.Run("numRows zero error", func(t *testing.T) {
		_, err := b.InsertRows(testFormatter, "users", []ColumnInfo{{Name: "id"}}, 0)
		assert.Error(t, err)
	})

	t.Run("numRows negative error", func(t *testing.T) {
		_, err := b.InsertRows(testFormatter, "users", []ColumnInfo{{Name: "id"}}, -1)
		assert.Error(t, err)
	})
}

func TestStdReturningQueryBuilder_InsertReturning(t *testing.T) {
	b := StdReturningQueryBuilder{}

	t.Run("returning single column", func(t *testing.T) {
		got, err := b.InsertReturning(testFormatter, "users", []ColumnInfo{
			{Name: "name"},
			{Name: "email"},
		}, "id")
		require.NoError(t, err)
		assert.Equal(t, `INSERT INTO users(name,email) VALUES($1,$2) RETURNING id`, got)
	})

	t.Run("returning star", func(t *testing.T) {
		got, err := b.InsertReturning(testFormatter, "users", []ColumnInfo{
			{Name: "name"},
		}, "*")
		require.NoError(t, err)
		assert.Equal(t, `INSERT INTO users(name) VALUES($1) RETURNING *`, got)
	})
}

func TestStdReturningQueryBuilder_UpdateReturning(t *testing.T) {
	b := StdReturningQueryBuilder{}

	t.Run("with where args", func(t *testing.T) {
		gotQuery, gotArgs, err := b.UpdateReturning(
			testFormatter, "users",
			Values{"name": "Alice"},
			"*",
			"id = $1", []any{42},
		)
		require.NoError(t, err)
		assert.Equal(t, `UPDATE users SET name=$2 WHERE id = $1 RETURNING *`, gotQuery)
		require.Len(t, gotArgs, 2)
		assert.Equal(t, 42, gotArgs[0])
		assert.Equal(t, "Alice", gotArgs[1])
	})

	t.Run("returning specific columns", func(t *testing.T) {
		gotQuery, _, err := b.UpdateReturning(
			testFormatter, "users",
			Values{"score": 100},
			"id, score",
			"active = true", nil,
		)
		require.NoError(t, err)
		assert.Equal(t, `UPDATE users SET score=$1 WHERE active = true RETURNING id, score`, gotQuery)
	})
}
