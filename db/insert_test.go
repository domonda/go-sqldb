package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestInsertStruct(t *testing.T) {
	type Struct1 struct {
		TableName `db:"my_table"`
		ID        int    `db:"id,pk"`
		Name      string `db:"name"`
	}

	tests := []struct {
		name      string
		rowStruct StructWithTableName
		options   []QueryOption
		conn      *sqldb.RecordingMockConn
		want      sqldb.MockConnRecording
		wantErr   bool
	}{
		{
			name: "simple",
			rowStruct: &Struct1{
				ID:   1,
				Name: "test",
			},
			conn: sqldb.NewRecordingMockConn("$", false),
			want: sqldb.MockConnRecording{
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
				ID   int    `db:"id,pk"`
				Name string `db:"name"`
			}{},
			conn:    sqldb.NewRecordingMockConn("$", false),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithConn(context.Background(), tt.conn)
			err := InsertStruct(ctx, tt.rowStruct, tt.options...)
			if tt.wantErr {
				require.Error(t, err, "error from InsertStructWithTableName")
				return
			}
			require.NoError(t, err, "error from InsertStructWithTableName")
			require.Equal(t, tt.want, tt.conn.MockConnRecording, "MockConnRecording")
		})
	}
}
