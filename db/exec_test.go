package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestExec(t *testing.T) {
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

		err := Exec(ctx, "DELETE FROM users WHERE id = $1", 42)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "DELETE FROM users WHERE id = $1", gotQuery)
		require.Equal(t, []any{42}, gotArgs)
	})

	t.Run("error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		testErr := errors.New("exec failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := Exec(ctx, "DELETE FROM users WHERE id = $1", 42)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}

func TestExecStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		var gotQueries []string
		var gotAllArgs [][]any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQueries = append(gotQueries, query)
			gotAllArgs = append(gotAllArgs, args)
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		execFunc, closeStmt, err := ExecStmt(ctx, "DELETE FROM users WHERE id = $1")
		require.NoError(t, err)
		defer closeStmt()

		err = execFunc(t.Context(), 1)
		require.NoError(t, err)

		err = execFunc(t.Context(), 2)
		require.NoError(t, err)

		require.Equal(t, 2, execCount, "MockExec call count")
		require.Equal(t, []any{1}, gotAllArgs[0])
		require.Equal(t, []any{2}, gotAllArgs[1])
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

		_, _, err := ExecStmt(ctx, "DELETE FROM users WHERE id = $1")
		require.ErrorIs(t, err, prepErr)
		require.Equal(t, 1, prepareCount, "MockPrepare call count")
	})

	t.Run("exec func error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var execCount int
		testErr := errors.New("exec failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		execFunc, closeStmt, err := ExecStmt(ctx, "DELETE FROM users WHERE id = $1")
		require.NoError(t, err)
		defer closeStmt()

		err = execFunc(t.Context(), 1)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}

