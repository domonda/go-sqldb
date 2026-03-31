package db

import (
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestQueryRowAs(t *testing.T) {
	query := /*sql*/ `SELECT EXISTS (SELECT FROM my_table WHERE id = $1)`
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
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
	ctx := testContext(t, mock)

	// id 666 has a row with the value true
	value, err := QueryRowAs[bool](ctx, query, 666)
	require.NoError(t, err)
	require.Equal(t, true, value, "QueryRowAs[bool] result")

	// id 777 has no rows
	value, err = QueryRowAs[bool](ctx, query, 777)
	require.ErrorIs(t, err, sql.ErrNoRows, "QueryRowAs[bool] result for 777 is sql.ErrNoRows")
}

func TestQueryRowAs2(t *testing.T) {
	query := /*sql*/ `SELECT id, name FROM my_table WHERE id = $1`
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"id", "name"},
			[][]driver.Value{{int64(1), "Alice"}},
			query,
			1,
		).
		WithQueryResult(
			[]string{"id", "name"},
			nil,
			query,
			999,
		)
	ctx := testContext(t, mock)

	t.Run("success", func(t *testing.T) {
		// int64 driver value can scan into int variable
		id, name, err := QueryRowAs2[int, string](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, err := QueryRowAs2[int, string](ctx, query, 999)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestQueryRowAs3(t *testing.T) {
	query := /*sql*/ `SELECT id, name, active FROM my_table WHERE id = $1`
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"id", "name", "active"},
			[][]driver.Value{{int64(1), "Alice", true}},
			query,
			1,
		).
		WithQueryResult(
			[]string{"id", "name", "active"},
			nil,
			query,
			999,
		)
	ctx := testContext(t, mock)

	t.Run("success", func(t *testing.T) {
		id, name, active, err := QueryRowAs3[int, string, bool](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
		require.Equal(t, true, active)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, _, err := QueryRowAs3[int, string, bool](ctx, query, 999)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestQueryRowAs4(t *testing.T) {
	query := /*sql*/ `SELECT id, name, active, score FROM my_table WHERE id = $1`
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"id", "name", "active", "score"},
			[][]driver.Value{{int64(1), "Alice", true, float64(9.5)}},
			query,
			1,
		).
		WithQueryResult(
			[]string{"id", "name", "active", "score"},
			nil,
			query,
			999,
		)
	ctx := testContext(t, mock)

	t.Run("success", func(t *testing.T) {
		id, name, active, score, err := QueryRowAs4[int, string, bool, float64](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
		require.Equal(t, true, active)
		require.Equal(t, float64(9.5), score)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, _, _, err := QueryRowAs4[int, string, bool, float64](ctx, query, 999)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestQueryRowAs5(t *testing.T) {
	query := /*sql*/ `SELECT id, name, active, score, label FROM my_table WHERE id = $1`
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"id", "name", "active", "score", "label"},
			[][]driver.Value{{int64(1), "Alice", true, float64(9.5), "admin"}},
			query,
			1,
		).
		WithQueryResult(
			[]string{"id", "name", "active", "score", "label"},
			nil,
			query,
			999,
		)
	ctx := testContext(t, mock)

	t.Run("success", func(t *testing.T) {
		id, name, active, score, label, err := QueryRowAs5[int, string, bool, float64, string](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
		require.Equal(t, true, active)
		require.Equal(t, float64(9.5), score)
		require.Equal(t, "admin", label)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, _, _, _, err := QueryRowAs5[int, string, bool, float64, string](ctx, query, 999)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestQueryRowAsOr(t *testing.T) {
	query := /*sql*/ `SELECT EXISTS (SELECT FROM my_table WHERE id = $1)`
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
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
	ctx := testContext(t, mock)

	// id 666 has a row with the value true
	value, err := QueryRowAsOr(ctx, false, query, 666)
	require.NoError(t, err)
	require.Equal(t, true, value, "QueryRowAsOr[bool] result for 666")

	// id 777 has no rows
	value, err = QueryRowAsOr(ctx, false, query, 777)
	require.NoError(t, err)
	require.Equal(t, false, value, "QueryRowAsOr[bool] result for 777")
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
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
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
	ctx := testContext(t, mock)
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
