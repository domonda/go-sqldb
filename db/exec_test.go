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

		err := Exec(ctx, "DELETE FROM users WHERE id = $1", 42)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "DELETE FROM users WHERE id = $1", gotQuery)
		require.Equal(t, []any{42}, gotArgs)
	})

	t.Run("error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		testErr := errors.New("exec failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		ctx := testContext(t, mock)

		err := Exec(ctx, "DELETE FROM users WHERE id = $1", 42)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}

func TestExecRowsAffected(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		mock.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 5, nil
		}
		ctx := testContext(t, mock)

		n, err := ExecRowsAffected(ctx, "UPDATE users SET active = $1", true)
		require.NoError(t, err)
		require.Equal(t, int64(5), n)
	})

	t.Run("error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		testErr := errors.New("exec failed")
		mock.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 0, testErr
		}
		ctx := testContext(t, mock)

		n, err := ExecRowsAffected(ctx, "UPDATE users SET active = $1", true)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, int64(0), n)
	})
}

func TestExecStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
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
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var prepareCount int
		prepErr := errors.New("prepare failed")
		mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		ctx := testContext(t, mock)

		_, _, err := ExecStmt(ctx, "DELETE FROM users WHERE id = $1")
		require.ErrorIs(t, err, prepErr)
		require.Equal(t, 1, prepareCount, "MockPrepare call count")
	})

	t.Run("exec func error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		testErr := errors.New("exec failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		ctx := testContext(t, mock)

		execFunc, closeStmt, err := ExecStmt(ctx, "DELETE FROM users WHERE id = $1")
		require.NoError(t, err)
		defer closeStmt()

		err = execFunc(t.Context(), 1)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}

func TestExecRowsAffectedStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		mock.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			return 3, nil
		}
		ctx := testContext(t, mock)

		execFunc, closeStmt, err := ExecRowsAffectedStmt(ctx, "UPDATE users SET active = $1")
		require.NoError(t, err)
		defer closeStmt()

		n, err := execFunc(t.Context(), true)
		require.NoError(t, err)
		require.Equal(t, int64(3), n)
		require.Equal(t, 1, execCount, "MockExecRowsAffected call count")
	})

	t.Run("multiple calls", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var callCount int
		mock.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			callCount++
			return int64(callCount), nil
		}
		ctx := testContext(t, mock)

		execFunc, closeStmt, err := ExecRowsAffectedStmt(ctx, "DELETE FROM users WHERE id = $1")
		require.NoError(t, err)
		defer closeStmt()

		n1, err := execFunc(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), n1)

		n2, err := execFunc(t.Context(), 2)
		require.NoError(t, err)
		require.Equal(t, int64(2), n2)

		require.Equal(t, 2, callCount, "MockExecRowsAffected call count")
	})

	t.Run("prepare error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		prepErr := errors.New("prepare failed")
		mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
			return nil, prepErr
		}
		ctx := testContext(t, mock)

		_, _, err := ExecRowsAffectedStmt(ctx, "UPDATE users SET active = $1")
		require.ErrorIs(t, err, prepErr)
	})

	t.Run("exec func error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		testErr := errors.New("exec failed")
		mock.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 0, testErr
		}
		ctx := testContext(t, mock)

		execFunc, closeStmt, err := ExecRowsAffectedStmt(ctx, "UPDATE users SET active = $1")
		require.NoError(t, err)
		defer closeStmt()

		n, err := execFunc(t.Context(), true)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, int64(0), n)
	})
}
