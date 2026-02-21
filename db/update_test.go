package db

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestUpdate(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
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

		err := Update(ctx, "users", sqldb.Values{"name": "Bob"}, "id = $1", 42)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "UPDATE users SET name=$2 WHERE id = $1", gotQuery)
		require.Equal(t, []any{42, "Bob"}, gotArgs)
	})

	t.Run("multiple values", func(t *testing.T) {
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

		err := Update(ctx, "users", sqldb.Values{"name": "Bob", "active": true}, "id = $1", 42)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		// Values are sorted alphabetically: active, name
		require.Equal(t, "UPDATE users SET active=$2, name=$3 WHERE id = $1", gotQuery)
		require.Equal(t, []any{42, true, "Bob"}, gotArgs)
	})

	t.Run("empty values error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := Update(ctx, "users", sqldb.Values{}, "id = $1", 42)
		require.Error(t, err)
	})

	t.Run("exec error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		testErr := errors.New("update failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := Update(ctx, "users", sqldb.Values{"name": "Bob"}, "id = $1", 42)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}

func TestUpdateRowStruct(t *testing.T) {
	type UserRow struct {
		ID     int    `db:"id,primarykey"`
		Name   string `db:"name"`
		Active bool   `db:"active"`
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

		err := UpdateRowStruct(ctx, "users", UserRow{ID: 1, Name: "Alice", Active: true})
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		// Columns in struct field order: id(PK), name, active
		// SET non-PK columns, WHERE PK columns
		require.Equal(t, "UPDATE users SET name=$2, active=$3 WHERE id = $1", gotQuery)
		require.Equal(t, []any{1, "Alice", true}, gotArgs)
	})

	t.Run("with pointer", func(t *testing.T) {
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

		err := UpdateRowStruct(ctx, "users", &UserRow{ID: 2, Name: "Bob", Active: false})
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "UPDATE users SET name=$2, active=$3 WHERE id = $1", gotQuery)
		require.Equal(t, []any{2, "Bob", false}, gotArgs)
	})

	t.Run("with IgnoreColumns option", func(t *testing.T) {
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

		err := UpdateRowStruct(ctx, "users", UserRow{ID: 1, Name: "Alice", Active: true}, sqldb.IgnoreColumns("active"))
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "UPDATE users SET name=$2 WHERE id = $1", gotQuery)
		require.Equal(t, []any{1, "Alice"}, gotArgs)
	})

	t.Run("nil struct error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpdateRowStruct(ctx, "users", (*UserRow)(nil))
		require.Error(t, err)
		require.Contains(t, err.Error(), "can't update nil")
	})

	t.Run("non-struct error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpdateRowStruct(ctx, "users", "not a struct")
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected struct")
	})

	t.Run("no primary key error", func(t *testing.T) {
		type NoPKRow struct {
			Name string `db:"name"`
		}
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpdateRowStruct(ctx, "users", NoPKRow{Name: "test"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no mapped primary key")
	})

	t.Run("exec error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		testErr := errors.New("update failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UpdateRowStruct(ctx, "users", UserRow{ID: 1, Name: "Alice"})
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}

// assertArgs is a test helper for comparing argument slices using fmt.Sprint
// to handle type differences (e.g., int vs int64).
func assertArgs(t *testing.T, got, want []any) {
	t.Helper()
	require.Equal(t, len(want), len(got), "args length")
	for i := range want {
		if fmt.Sprint(got[i]) != fmt.Sprint(want[i]) {
			t.Errorf("args[%d] = %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
		}
	}
}
