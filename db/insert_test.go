package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestInsertStructWithTableName(t *testing.T) {
	type S struct {
		TableName `db:"my_table"`

		ID   int    `db:"id,pk"`
		Name string `db:"name"`
	}

	tests := []struct {
		name          string
		rowStruct     StructWithTableName
		ignoreColumns []ColumnFilter
		conn          *sqldb.RecordingMockConn
		want          sqldb.MockConnRecording
		wantErr       bool
	}{
		{
			name: "simple",
			rowStruct: &S{
				ID:   1,
				Name: "test",
			},
			conn: sqldb.NewRecordingMockConn("$%d", false),
			want: sqldb.MockConnRecording{
				Execs: []sqldb.QueryData{
					{Query: "INSERT INTO my_table(id,name) VALUES($1,$2)", Args: []any{1, "test"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithConn(context.Background(), tt.conn)
			err := InsertStructWithTableName(ctx, tt.rowStruct, tt.ignoreColumns...)
			if tt.wantErr {
				require.Error(t, err, "error from InsertStructWithTableName")
				return
			}
			require.NoError(t, err, "error from InsertStructWithTableName")
			require.Equal(t, tt.want, tt.conn.MockConnRecording, "MockConnRecording")
		})
	}
}
