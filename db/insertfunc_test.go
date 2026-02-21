package db

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestInsertUnique(t *testing.T) {
	t.Run("inserted", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			// InsertUnique checks rows.Next() — return a row to indicate insertion
			return sqldb.NewMockRows([]string{"bool"}, [][]driver.Value{{true}})
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		inserted, err := InsertUnique(ctx, "users", sqldb.Values{"id": 1, "name": "Alice"}, "(id)")
		require.NoError(t, err)
		require.True(t, inserted)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		// Values sorted alphabetically: id, name
		require.Equal(t, "INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id) DO NOTHING RETURNING TRUE", gotQuery)
		require.Equal(t, []any{1, "Alice"}, gotArgs)
	})

	t.Run("not inserted (conflict)", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			// No rows returned means conflict — not inserted
			return sqldb.NewMockRows([]string{"bool"}, nil)
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		inserted, err := InsertUnique(ctx, "users", sqldb.Values{"id": 1, "name": "Alice"}, "(id)")
		require.NoError(t, err)
		require.False(t, inserted)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})

	t.Run("query error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var queryCount int
		testErr := errors.New("insert failed")
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewErrRows(testErr)
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

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
		mock := sqldb.NewMockConn("$", nil, nil)
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return sqldb.NewMockRows([]string{"bool"}, [][]driver.Value{{true}})
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		inserted, err := InsertUniqueRowStruct(ctx, &UserRow{ID: 1, Name: "Alice"}, "(id)")
		require.NoError(t, err)
		require.True(t, inserted)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, "INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id) DO NOTHING RETURNING TRUE", gotQuery)
		require.Equal(t, []any{1, "Alice"}, gotArgs)
	})

	t.Run("with IgnoreColumns", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return sqldb.NewMockRows([]string{"bool"}, [][]driver.Value{{true}})
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

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
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		var gotAllArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotAllArgs = append(gotAllArgs, args)
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

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
		mock := sqldb.NewMockConn("$", nil, nil)
		var prepareCount int
		prepErr := errors.New("prepare failed")
		mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

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

	t.Run("success multiple rows", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		var gotAllArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotAllArgs = append(gotAllArgs, args)
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		rows := []ItemRow{
			{ID: 1, Name: "Item1"},
			{ID: 2, Name: "Item2"},
			{ID: 3, Name: "Item3"},
		}
		err := InsertRowStructs(ctx, rows)
		require.NoError(t, err)
		require.Equal(t, 3, execCount, "MockExec call count")
		require.Equal(t, []any{1, "Item1"}, gotAllArgs[0])
		require.Equal(t, []any{2, "Item2"}, gotAllArgs[1])
		require.Equal(t, []any{3, "Item3"}, gotAllArgs[2])
	})

	t.Run("empty slice", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := InsertRowStructs(ctx, []ItemRow{})
		require.NoError(t, err)
	})

	t.Run("exec error on second row", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		testErr := errors.New("insert row 2 failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			if execCount == 2 {
				return testErr
			}
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		rows := []ItemRow{
			{ID: 1, Name: "Item1"},
			{ID: 2, Name: "Item2"},
		}
		err := InsertRowStructs(ctx, rows)
		require.ErrorIs(t, err, testErr)
	})
}
