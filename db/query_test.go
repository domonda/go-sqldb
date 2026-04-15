package db_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
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
	value, err := db.QueryRowAs[bool](ctx, query, 666)
	require.NoError(t, err)
	require.Equal(t, true, value, "QueryRowAs[bool] result")

	// id 777 has no rows
	value, err = db.QueryRowAs[bool](ctx, query, 777)
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
		id, name, err := db.QueryRowAs2[int, string](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, err := db.QueryRowAs2[int, string](ctx, query, 999)
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
		id, name, active, err := db.QueryRowAs3[int, string, bool](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
		require.Equal(t, true, active)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, _, err := db.QueryRowAs3[int, string, bool](ctx, query, 999)
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
		id, name, active, score, err := db.QueryRowAs4[int, string, bool, float64](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
		require.Equal(t, true, active)
		require.Equal(t, float64(9.5), score)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, _, _, err := db.QueryRowAs4[int, string, bool, float64](ctx, query, 999)
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
		id, name, active, score, label, err := db.QueryRowAs5[int, string, bool, float64, string](ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, "Alice", name)
		require.Equal(t, true, active)
		require.Equal(t, float64(9.5), score)
		require.Equal(t, "admin", label)
	})

	t.Run("no rows", func(t *testing.T) {
		_, _, _, _, _, err := db.QueryRowAs5[int, string, bool, float64, string](ctx, query, 999)
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
	value, err := db.QueryRowAsOr(ctx, false, query, 666)
	require.NoError(t, err)
	require.Equal(t, true, value, "QueryRowAsOr[bool] result for 666")

	// id 777 has no rows
	value, err = db.QueryRowAsOr(ctx, false, query, 777)
	require.NoError(t, err)
	require.Equal(t, false, value, "QueryRowAsOr[bool] result for 777")
}

func TestQueryRowAsStrings(t *testing.T) {
	query := /*sql*/ `SELECT id, name, active, data, missing FROM my_table WHERE id = $1`
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"id", "name", "active", "data", "missing"},
			[][]driver.Value{{int64(7), "Alice", true, []byte("raw"), nil}},
			query,
			7,
		).
		WithQueryResult(
			[]string{"id", "name", "active", "data", "missing"},
			nil,
			query,
			999,
		)
	ctx := testContext(t, mock)

	t.Run("success", func(t *testing.T) {
		vals, err := db.QueryRowAsStrings(ctx, query, 7)
		require.NoError(t, err)
		require.Equal(t, []string{"7", "Alice", "true", "raw", ""}, vals)
	})

	t.Run("no rows", func(t *testing.T) {
		_, err := db.QueryRowAsStrings(ctx, query, 999)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestQueryRowAsStringsWithHeader(t *testing.T) {
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
		rows, err := db.QueryRowAsStringsWithHeader(ctx, query, 1)
		require.NoError(t, err)
		require.Equal(t, [][]string{
			{"id", "name", "active"},
			{"1", "Alice", "true"},
		}, rows)
	})

	t.Run("no rows", func(t *testing.T) {
		_, err := db.QueryRowAsStringsWithHeader(ctx, query, 999)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func ExampleQueryRowsAsMapSlice() {
	// A typical use of QueryRowsAsMapSlice is to turn a query result
	// into a JSON array of objects keyed by column name. The mock connection
	// below stands in for a real database.
	createdAt := time.Date(2026, 4, 14, 9, 30, 0, 0, time.UTC)
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
		return sqldb.NewMockRows("id", "name", "data", "created_at").
			WithRow(int64(1), "Alice", []byte("hello"), createdAt).
			WithRow(int64(2), "Bob", []byte{0xff, 0xfe}, createdAt.Add(time.Hour))
	}
	ctx := db.ContextWithConn(context.Background(), mock)

	// BytesToStringScanConverter turns []byte columns into plain strings
	// (or hex with the given prefix for non-UTF-8), otherwise
	// json.MarshalIndent would base64-encode them.
	// TimeToStringScanConverter formats time.Time columns with the given
	// layout instead of relying on time.Time's default JSON encoding.
	// Multiple converters are combined with db.ScanConverters.
	rows, err := db.QueryRowsAsMapSlice(ctx,
		db.ScanConverters{
			db.BytesToStringScanConverter(`\x`),
			db.TimeToStringScanConverter(time.DateTime),
		},
		`SELECT id, name, data, created_at FROM users`,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	out, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(out))

	// Output:
	// [
	//   {
	//     "created_at": "2026-04-14 09:30:00",
	//     "data": "hello",
	//     "id": 1,
	//     "name": "Alice"
	//   },
	//   {
	//     "created_at": "2026-04-14 10:30:00",
	//     "data": "\\xFFFE",
	//     "id": 2,
	//     "name": "Bob"
	//   }
	// ]
}

func TestQueryRowsAsMapSlice(t *testing.T) {
	query := /*sql*/ `SELECT id, name, data FROM my_table WHERE active = $1`

	t.Run("without converters", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
			WithQueryResult(
				[]string{"id", "name", "data"},
				[][]driver.Value{
					{int64(1), "Alice", []byte("hello")},
					{int64(2), "Bob", []byte{0xff, 0xfe}},
				},
				query,
				true,
			)
		ctx := testContext(t, mock)

		rows, err := db.QueryRowsAsMapSlice(ctx, nil, query, true)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		require.Equal(t, int64(1), rows[0]["id"])
		require.Equal(t, "Alice", rows[0]["name"])
		require.Equal(t, []byte("hello"), rows[0]["data"])
	})

	t.Run("with bytes converter", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
			WithQueryResult(
				[]string{"id", "name", "data"},
				[][]driver.Value{
					{int64(1), "Alice", []byte("hello")},
					{int64(2), "Bob", []byte{0xff, 0xfe}},
				},
				query,
				true,
			)
		ctx := testContext(t, mock)

		rows, err := db.QueryRowsAsMapSlice(ctx, db.BytesToStringScanConverter(`\x`), query, true)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		require.Equal(t, "hello", rows[0]["data"])
		require.Equal(t, `\xFFFE`, rows[1]["data"])
	})

	t.Run("context maxNumRows cap exceeded", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
			WithQueryResult(
				[]string{"id", "name", "data"},
				[][]driver.Value{
					{int64(1), "Alice", []byte("hello")},
					{int64(2), "Bob", []byte("world")},
					{int64(3), "Charlie", []byte("hi")},
				},
				query,
				true,
			)
		ctx := db.ContextWithMaxNumRows(testContext(t, mock), 2)

		rows, err := db.QueryRowsAsMapSlice(ctx, nil, query, true)
		var maxErr db.ErrMaxNumRowsExceeded
		require.ErrorAs(t, err, &maxErr)
		require.Equal(t, 2, maxErr.MaxNumRows)
		require.Len(t, rows, 2)
		require.Equal(t, int64(1), rows[0]["id"])
		require.Equal(t, int64(2), rows[1]["id"])
	})
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
			gotRows, err := db.QueryRowsAsStrings(ctx, tt.query, tt.args...)
			if tt.wantErr {
				require.Error(t, err, "QueryStrings() error")
				return
			}
			require.NoError(t, err, "QueryStrings() error")
			require.Equal(t, tt.wantRows, gotRows, "QueryStrings() result")
		})
	}
}
