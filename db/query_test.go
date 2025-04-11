package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func Test_isNonSQLScannerStruct(t *testing.T) {
	tests := []struct {
		t    reflect.Type
		want bool
	}{
		// Structs that do not implement sql.Scanner
		{t: reflect.TypeFor[struct{ X int }](), want: true},

		// Structs that implement sql.Scanner
		{t: reflect.TypeFor[time.Time](), want: false},
		{t: reflect.TypeFor[sql.NullTime](), want: false},

		// Non struct types
		{t: reflect.TypeFor[int](), want: false},
		{t: reflect.TypeFor[string](), want: false},
		{t: reflect.TypeFor[[]byte](), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.t.String(), func(t *testing.T) {
			if got := isNonSQLScannerStruct(tt.t); got != tt.want {
				t.Errorf("isNonSQLScannerStruct(%s) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestQueryValue(t *testing.T) {
	query := /*sql*/ `SELECT EXISTS (SELECT FROM my_table WHERE id = $1)`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"exists"},       // columns
			[][]driver.Value{{true}}, // rows
			query,                    // query
			666,                      // args
		).
		WithQueryResult(
			[]string{"exists"}, // columns
			nil,                // rows
			query,              // query
			777,                // args
		)
	ctx := ContextWithConn(context.Background(), conn)

	// id 666 has a row with the value true
	value, err := QueryValue[bool](ctx, query, 666)
	require.NoError(t, err)
	require.Equal(t, true, value, "QueryValue[bool] result")

	// id 777 has no rows
	value, err = QueryValue[bool](ctx, query, 777)
	require.ErrorIs(t, err, sql.ErrNoRows, "QueryValue[bool] result for 777 is sql.ErrNoRows")
}

func TestQueryValueOr(t *testing.T) {
	query := /*sql*/ `SELECT EXISTS (SELECT FROM my_table WHERE id = $1)`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"exists"},       // columns
			[][]driver.Value{{true}}, // rows
			query,                    // query
			666,                      // args
		).
		WithQueryResult(
			[]string{"exists"}, // columns
			nil,                // rows
			query,              // query
			777,                // args
		)
	ctx := ContextWithConn(context.Background(), conn)

	// id 666 has a row with the value true
	value, err := QueryValueOr(ctx, false, query, 666)
	require.NoError(t, err)
	require.Equal(t, true, value, "QueryValueOr[bool] result for 666")

	// id 777 has no rows
	value, err = QueryValueOr(ctx, false, query, 777)
	require.NoError(t, err)
	require.Equal(t, false, value, "QueryValueOr[bool] result for 777")
}

func TestQueryStrings(t *testing.T) {
	query := /*sql*/ `SELECT test_no, col1, col2, col3 FROM my_table WHERE test_no = $1`
	tests := []struct {
		name     string
		query    string
		args     []any
		wantRows [][]string
		wantErr  bool
	}{
		{
			name:  "test_no 0: no rows",
			query: query,
			args:  []any{0},
			wantRows: [][]string{
				{"test_no", "col1", "col2", "col3"},
			},
		},
		{
			name:  "test_no 1: 3 rows",
			query: query,
			args:  []any{1},
			wantRows: [][]string{
				{"test_no", "col1", "col2", "col3"},
				{"1", "row0_col1", "row0_col2", "2025-01-02 03:04:05 +0000 UTC"},
				{"1", "row1_col1", "", "0001-01-01 00:00:00 +0000 UTC"},
				{"1", "row2_col1", "bytes", "2025-01-02 03:04:05 +0000 UTC"},
			},
		},
	}
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"test_no", "col1", "col2", "col3"},
			[][]driver.Value{},
			query,
			0,
		).
		WithQueryResult(
			[]string{"test_no", "col1", "col2", "col3"},
			[][]driver.Value{
				{int64(1), "row0_col1", "row0_col2", "2025-01-02 03:04:05 +0000 UTC"},
				{int64(1), "row1_col1", nil, time.Time{}},
				{int64(1), "row2_col1", []byte("bytes"), time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)},
			},
			query,
			1,
		)
	ctx := ContextWithConn(context.Background(), conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRows, err := QueryRowsAsStrings(ctx, tt.query, tt.args...)
			if tt.wantErr {
				require.Error(t, err, "QueryStrings() error")
				return
			}
			require.NoError(t, err, "QueryStrings() error")
			require.Equal(t, tt.wantRows, gotRows, "QueryStrings() result")
		})
	}
}
