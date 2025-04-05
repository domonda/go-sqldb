package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestInsertRowStruct(t *testing.T) {
	type Struct1 struct {
		TableName `db:"my_table"`
		ID        int    `db:"id"`
		Name      string `db:"name"`
	}

	tests := []struct {
		name      string
		rowStruct StructWithTableName
		options   []QueryOption
		conn      *sqldb.MockConn
		want      sqldb.QueryRecordings
		wantErr   bool
	}{
		{
			name: "simple",
			rowStruct: &Struct1{
				ID:   1,
				Name: "test",
			},
			conn: sqldb.NewMockConn("$", nil, os.Stdout),
			want: sqldb.QueryRecordings{
				Execs: []sqldb.QueryData{
					{Query: "INSERT INTO my_table(id,name) VALUES($1,$2)", Args: []any{1, "test"}},
				},
			},
		},
		// Error cases
		{
			name: "TableName without name tag",
			rowStruct: struct {
				TableName
				ID   int    `db:"id"`
				Name string `db:"name"`
			}{},
			conn:    sqldb.NewMockConn("$", nil, os.Stdout),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithConn(context.Background(), tt.conn)
			err := InsertRowStruct(ctx, tt.rowStruct, tt.options...)
			if tt.wantErr {
				require.Error(t, err, "error from InsertStruct")
				return
			}
			require.NoError(t, err, "error from InsertStruct")
			require.Equal(t, tt.want, tt.conn.Recordings, "MockConn.Recordings")
		})
	}
}

func TestInsert(t *testing.T) {
	timestamp := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		table   string
		values  Values
		conn    *sqldb.MockConn
		want    sqldb.QueryRecordings
		wantErr bool
	}{
		{
			name:  "basic",
			table: "public.my_table",
			values: Values{
				"id":         1,
				"name":       "Test",
				"created_at": timestamp,
				"updated_at": sql.NullTime{},
			},
			conn: sqldb.NewMockConn("$", nil, os.Stdout),
			want: sqldb.QueryRecordings{
				Execs: []sqldb.QueryData{
					{
						Query: `INSERT INTO public.my_table(created_at,id,name,updated_at) VALUES($1,$2,$3,$4)`,
						Args:  []any{timestamp, 1, "Test", sql.NullTime{}},
					},
				},
			},
		},

		// Error cases
		{
			name:    "no values",
			table:   "public.my_table",
			values:  Values{},
			conn:    sqldb.NewMockConn("$", nil, os.Stdout),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithConn(context.Background(), tt.conn)
			err := Insert(ctx, tt.table, tt.values)
			if tt.wantErr {
				require.Error(t, err, "error from Insert")
				return
			}
			require.NoError(t, err, "error from Insert")
			require.Equal(t, tt.want, tt.conn.Recordings, "MockConn.Recordings")
		})
	}
}
