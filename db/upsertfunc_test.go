package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestUpsertRowStruct(t *testing.T) {
	type UserRow struct {
		sqldb.TableName `db:"users"`
		ID              int    `db:"id,primarykey"`
		Name            string `db:"name"`
		Active          bool   `db:"active"`
	}

	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpsertRowStruct(ctx, &UserRow{ID: 1, Name: "Alice", Active: true})
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "INSERT INTO users(id,name,active) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, active=$3", gotQuery)
		require.Equal(t, []any{1, "Alice", true}, gotArgs)
	})

	t.Run("with IgnoreColumns", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpsertRowStruct(ctx, &UserRow{ID: 1, Name: "Alice", Active: true}, sqldb.IgnoreColumns("active"))
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET name=$2", gotQuery)
		require.Equal(t, []any{1, "Alice"}, gotArgs)
	})

	t.Run("exec error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		testErr := errors.New("upsert failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpsertRowStruct(ctx, &UserRow{ID: 1, Name: "Alice"})
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})

	t.Run("nil struct error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpsertRowStruct(ctx, (*UserRow)(nil))
		require.Error(t, err)
	})
}

func TestUpsertRowStructs(t *testing.T) {
	type ItemRow struct {
		sqldb.TableName `db:"items"`
		ID              int    `db:"id,primarykey"`
		Name            string `db:"name"`
	}

	t.Run("success multiple rows", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		var queries []string
		var allArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			queries = append(queries, query)
			allArgs = append(allArgs, args)
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		rows := []ItemRow{
			{ID: 1, Name: "Item1"},
			{ID: 2, Name: "Item2"},
		}
		err := UpsertRowStructs(ctx, rows)
		require.NoError(t, err)
		require.Equal(t, 2, execCount, "MockExec call count")
		for _, q := range queries {
			require.Equal(t, "INSERT INTO items(id,name) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET name=$2", q)
		}
		require.Equal(t, []any{1, "Item1"}, allArgs[0])
		require.Equal(t, []any{2, "Item2"}, allArgs[1])
	})

	t.Run("empty slice", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpsertRowStructs(ctx, []ItemRow{})
		require.NoError(t, err)
	})

	t.Run("exec error on second row", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		testErr := errors.New("upsert row 2 failed")
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
		err := UpsertRowStructs(ctx, rows)
		require.ErrorIs(t, err, testErr)
	})
}
