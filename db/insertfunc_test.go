package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestInsertUnique(t *testing.T) {
	t.Run("inserted", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			// InsertUnique checks rows.Next() — return a row to indicate insertion
			return sqldb.NewMockRows("bool").WithRow(true)
		}
		ctx := testContext(t, mock)

		inserted, err := InsertUnique(ctx, "users", sqldb.Values{"id": 1, "name": "Alice"}, "(id)")
		require.NoError(t, err)
		require.True(t, inserted)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		// Values sorted alphabetically: id, name
		require.Equal(t, "INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id) DO NOTHING RETURNING TRUE", gotQuery)
		require.Equal(t, []any{1, "Alice"}, gotArgs)
	})

	t.Run("not inserted (conflict)", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			// No rows returned means conflict — not inserted
			return sqldb.NewMockRows("bool")
		}
		ctx := testContext(t, mock)

		inserted, err := InsertUnique(ctx, "users", sqldb.Values{"id": 1, "name": "Alice"}, "(id)")
		require.NoError(t, err)
		require.False(t, inserted)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})

	t.Run("query error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		testErr := errors.New("insert failed")
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewErrRows(testErr)
		}
		ctx := testContext(t, mock)

		_, err := InsertUnique(ctx, "users", sqldb.Values{"id": 1}, "(id)")
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})
}

func TestInsertUniqueRowStruct(t *testing.T) {
	type UserRow struct {
		sqldb.TableName `db:"users"`
		ID              int    `db:"id,primarykey"`
		Name            string `db:"name"`
	}

	t.Run("inserted", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return sqldb.NewMockRows("bool").WithRow(true)
		}
		ctx := testContext(t, mock)

		inserted, err := InsertUniqueRowStruct(ctx, &UserRow{ID: 1, Name: "Alice"}, "(id)")
		require.NoError(t, err)
		require.True(t, inserted)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, "INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id) DO NOTHING RETURNING TRUE", gotQuery)
		require.Equal(t, []any{1, "Alice"}, gotArgs)
	})

	t.Run("with IgnoreColumns", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return sqldb.NewMockRows("bool").WithRow(true)
		}
		ctx := testContext(t, mock)

		inserted, err := InsertUniqueRowStruct(ctx, &UserRow{ID: 1, Name: "Alice"}, "(id)", sqldb.IgnoreColumns("name"))
		require.NoError(t, err)
		require.True(t, inserted)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, "INSERT INTO users(id) VALUES($1) ON CONFLICT (id) DO NOTHING RETURNING TRUE", gotQuery)
		require.Equal(t, []any{1}, gotArgs)
	})
}

func TestInsertRowStructStmt(t *testing.T) {
	type ItemRow struct {
		sqldb.TableName `db:"items"`
		ID              int    `db:"id"`
		Name            string `db:"name"`
	}

	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		var gotAllArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotAllArgs = append(gotAllArgs, args)
			return nil
		}
		ctx := testContext(t, mock)

		insertFunc, closeStmt, err := InsertRowStructStmt[ItemRow](ctx)
		require.NoError(t, err)
		defer closeStmt()

		err = insertFunc(t.Context(), ItemRow{ID: 1, Name: "Item1"})
		require.NoError(t, err)

		err = insertFunc(t.Context(), ItemRow{ID: 2, Name: "Item2"})
		require.NoError(t, err)

		require.Equal(t, 2, execCount, "MockExec call count")
		require.Equal(t, []any{1, "Item1"}, gotAllArgs[0])
		require.Equal(t, []any{2, "Item2"}, gotAllArgs[1])
	})

	t.Run("prepare error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var prepareCount int
		prepErr := errors.New("prepare failed")
		mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		ctx := testContext(t, mock)

		_, _, err := InsertRowStructStmt[ItemRow](ctx)
		require.ErrorIs(t, err, prepErr)
		require.Equal(t, 1, prepareCount, "MockPrepare call count")
	})
}

func TestInsertRowStructs(t *testing.T) {
	type ItemRow struct {
		sqldb.TableName `db:"items"`
		ID              int    `db:"id"`
		Name            string `db:"name"`
	}

	t.Run("empty slice", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		ctx := testContext(t, mock)

		err := InsertRowStructs(ctx, []ItemRow{})
		require.NoError(t, err)
	})

	t.Run("single row delegates to InsertRowStruct", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		err := InsertRowStructs(ctx, []ItemRow{{ID: 1, Name: "Item1"}})
		require.NoError(t, err)
		require.Equal(t, "INSERT INTO items(id,name) VALUES($1,$2)", gotQuery)
		require.Equal(t, []any{1, "Item1"}, gotArgs)
	})

	t.Run("all rows fit single batch", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		rows := []ItemRow{
			{ID: 1, Name: "Item1"},
			{ID: 2, Name: "Item2"},
			{ID: 3, Name: "Item3"},
		}
		err := InsertRowStructs(ctx, rows)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "single multi-row INSERT")
		require.Equal(t, "INSERT INTO items(id,name) VALUES($1,$2),($3,$4),($5,$6)", gotQuery)
		require.Equal(t, []any{1, "Item1", 2, "Item2", 3, "Item3"}, gotArgs)
	})

	t.Run("exec error single batch", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		testErr := errors.New("insert failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			return testErr
		}
		ctx := testContext(t, mock)

		rows := []ItemRow{
			{ID: 1, Name: "Item1"},
			{ID: 2, Name: "Item2"},
		}
		err := InsertRowStructs(ctx, rows)
		require.ErrorIs(t, err, testErr)
	})

	t.Run("multiple full batches uses prepare", func(t *testing.T) {
		// MockMaxArgs=4, 2 cols → rowsPerBatch=2
		// 4 rows → 2 full batches, no remainder → uses prepare
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		mock.MockMaxArgs = 4
		var execCount int
		var gotQueries []string
		var gotAllArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQueries = append(gotQueries, query)
			gotAllArgs = append(gotAllArgs, args)
			return nil
		}
		ctx := testContext(t, mock)

		rows := []ItemRow{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
			{ID: 3, Name: "C"},
			{ID: 4, Name: "D"},
		}
		err := InsertRowStructs(ctx, rows)
		require.NoError(t, err)
		require.Equal(t, 2, execCount, "2 full batch executions")
		batchQuery := "INSERT INTO items(id,name) VALUES($1,$2),($3,$4)"
		require.Equal(t, batchQuery, gotQueries[0])
		require.Equal(t, batchQuery, gotQueries[1])
		require.Equal(t, []any{1, "A", 2, "B"}, gotAllArgs[0])
		require.Equal(t, []any{3, "C", 4, "D"}, gotAllArgs[1])
	})

	t.Run("full batches plus remainder", func(t *testing.T) {
		// MockMaxArgs=4, 2 cols → rowsPerBatch=2
		// 5 rows → 2 full batches + 1 remainder row
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		mock.MockMaxArgs = 4
		var execCount int
		var gotQueries []string
		var gotAllArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQueries = append(gotQueries, query)
			gotAllArgs = append(gotAllArgs, args)
			return nil
		}
		ctx := testContext(t, mock)

		rows := []ItemRow{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
			{ID: 3, Name: "C"},
			{ID: 4, Name: "D"},
			{ID: 5, Name: "E"},
		}
		err := InsertRowStructs(ctx, rows)
		require.NoError(t, err)
		require.Equal(t, 3, execCount, "2 full batches + 1 remainder")
		batchQuery := "INSERT INTO items(id,name) VALUES($1,$2),($3,$4)"
		remainQuery := "INSERT INTO items(id,name) VALUES($1,$2)"
		require.Equal(t, batchQuery, gotQueries[0])
		require.Equal(t, batchQuery, gotQueries[1])
		require.Equal(t, remainQuery, gotQueries[2])
		require.Equal(t, []any{1, "A", 2, "B"}, gotAllArgs[0])
		require.Equal(t, []any{3, "C", 4, "D"}, gotAllArgs[1])
		require.Equal(t, []any{5, "E"}, gotAllArgs[2])
	})

	t.Run("single full batch plus remainder no prepare", func(t *testing.T) {
		// MockMaxArgs=4, 2 cols → rowsPerBatch=2
		// 3 rows → 1 full batch + 1 remainder → no prepare (numFullBatches == 1)
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		mock.MockMaxArgs = 4
		var execCount int
		var gotQueries []string
		var gotAllArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQueries = append(gotQueries, query)
			gotAllArgs = append(gotAllArgs, args)
			return nil
		}
		ctx := testContext(t, mock)

		rows := []ItemRow{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
			{ID: 3, Name: "C"},
		}
		err := InsertRowStructs(ctx, rows)
		require.NoError(t, err)
		require.Equal(t, 2, execCount, "1 full batch + 1 remainder")
		require.Equal(t, "INSERT INTO items(id,name) VALUES($1,$2),($3,$4)", gotQueries[0])
		require.Equal(t, "INSERT INTO items(id,name) VALUES($1,$2)", gotQueries[1])
		require.Equal(t, []any{1, "A", 2, "B"}, gotAllArgs[0])
		require.Equal(t, []any{3, "C"}, gotAllArgs[1])
	})

	t.Run("exact single batch with low MaxArgs", func(t *testing.T) {
		// MockMaxArgs=4, 2 cols → rowsPerBatch=2
		// 2 rows → fits exactly in 1 batch → no transaction, no prepare
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		mock.MockMaxArgs = 4
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		rows := []ItemRow{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
		}
		err := InsertRowStructs(ctx, rows)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "single batch exec")
		require.Equal(t, "INSERT INTO items(id,name) VALUES($1,$2),($3,$4)", gotQuery)
		require.Equal(t, []any{1, "A", 2, "B"}, gotArgs)
	})
}
