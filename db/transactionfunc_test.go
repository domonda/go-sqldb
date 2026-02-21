package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestContextWithoutTransactions(t *testing.T) {
	ctx := t.Context()
	require.False(t, IsContextWithoutTransactions(ctx))

	ctx = ContextWithoutTransactions(ctx)
	require.True(t, IsContextWithoutTransactions(ctx))
}

func TestIsTransaction(t *testing.T) {
	t.Run("not in transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		require.False(t, IsTransaction(ctx))
	})

	t.Run("in transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1 // simulate active transaction
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		require.True(t, IsTransaction(ctx))
	})

	t.Run("in transaction but ContextWithoutTransactions", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1 // simulate active transaction
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithoutTransactions(ctx)

		require.False(t, IsTransaction(ctx))
	})
}

func TestValidateWithinTransaction(t *testing.T) {
	t.Run("within transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := ValidateWithinTransaction(ctx)
		require.NoError(t, err)
	})

	t.Run("not within transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := ValidateWithinTransaction(ctx)
		require.ErrorIs(t, err, sqldb.ErrNotWithinTransaction)
	})

	t.Run("ContextWithoutTransactions", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithoutTransactions(ctx)

		err := ValidateWithinTransaction(ctx)
		require.ErrorIs(t, err, sqldb.ErrNotWithinTransaction)
	})
}

func TestValidateNotWithinTransaction(t *testing.T) {
	t.Run("not within transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := ValidateNotWithinTransaction(ctx)
		require.NoError(t, err)
	})

	t.Run("within transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := ValidateNotWithinTransaction(ctx)
		require.ErrorIs(t, err, sqldb.ErrWithinTransaction)
	})

	t.Run("ContextWithoutTransactions bypasses check", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithoutTransactions(ctx)

		err := ValidateNotWithinTransaction(ctx)
		require.NoError(t, err)
	})
}

func TestTransaction_DB(t *testing.T) {
	t.Run("success commits", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var commitCount int
		mock.MockCommit = func() error {
			commitCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		var called bool
		err := Transaction(ctx, func(ctx context.Context) error {
			called = true
			require.True(t, IsTransaction(ctx), "should be in transaction")
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
		require.Equal(t, 1, commitCount, "MockCommit call count")
	})

	t.Run("error rolls back", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var rollbackCount int
		mock.MockRollback = func() error {
			rollbackCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		txErr := errors.New("tx failed")
		err := Transaction(ctx, func(ctx context.Context) error {
			return txErr
		})
		require.ErrorIs(t, err, txErr)
		require.Equal(t, 1, rollbackCount, "MockRollback call count")
	})

	t.Run("already in transaction passes through", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1 // simulate active transaction
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		var called bool
		err := Transaction(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("ContextWithoutTransactions bypasses transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithoutTransactions(ctx)

		var called bool
		err := Transaction(ctx, func(ctx context.Context) error {
			called = true
			// Should NOT be in a transaction
			require.False(t, IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})
}

func TestTransactionResult_DB(t *testing.T) {
	t.Run("success with result", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var commitCount int
		mock.MockCommit = func() error {
			commitCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		result, err := TransactionResult(ctx, func(ctx context.Context) (string, error) {
			return "hello", nil
		})
		require.NoError(t, err)
		require.Equal(t, "hello", result)
		require.Equal(t, 1, commitCount, "MockCommit call count")
	})

	t.Run("error returns zero result", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var rollbackCount int
		mock.MockRollback = func() error {
			rollbackCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		txErr := errors.New("tx failed")
		result, err := TransactionResult(ctx, func(ctx context.Context) (int, error) {
			return 0, txErr
		})
		require.ErrorIs(t, err, txErr)
		require.Equal(t, 0, result)
		require.Equal(t, 1, rollbackCount, "MockRollback call count")
	})
}

func TestIsolatedTransaction_DB(t *testing.T) {
	t.Run("success commits", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var commitCount int
		mock.MockCommit = func() error {
			commitCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := IsolatedTransaction(ctx, func(ctx context.Context) error {
			require.True(t, IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, commitCount, "MockCommit call count")
	})

	t.Run("error rolls back", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var rollbackCount int
		mock.MockRollback = func() error {
			rollbackCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		txErr := errors.New("isolated tx failed")
		err := IsolatedTransaction(ctx, func(ctx context.Context) error {
			return txErr
		})
		require.ErrorIs(t, err, txErr)
		require.Equal(t, 1, rollbackCount, "MockRollback call count")
	})

	t.Run("ContextWithoutTransactions bypasses transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithoutTransactions(ctx)

		var called bool
		err := IsolatedTransaction(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})
}

func TestOptionalTransaction_DB(t *testing.T) {
	t.Run("with transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var commitCount int
		mock.MockCommit = func() error {
			commitCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		var inTx bool
		err := OptionalTransaction(ctx, true, func(ctx context.Context) error {
			inTx = IsTransaction(ctx)
			return nil
		})
		require.NoError(t, err)
		require.True(t, inTx, "should be in transaction")
		require.Equal(t, 1, commitCount, "MockCommit call count")
	})

	t.Run("without transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		var inTx bool
		err := OptionalTransaction(ctx, false, func(ctx context.Context) error {
			inTx = IsTransaction(ctx)
			return nil
		})
		require.NoError(t, err)
		require.False(t, inTx, "should not be in transaction")
	})
}

func TestTransactionReadOnly_DB(t *testing.T) {
	t.Run("success commits", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var commitCount int
		mock.MockCommit = func() error {
			commitCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := TransactionReadOnly(ctx, func(ctx context.Context) error {
			require.True(t, IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, commitCount, "MockCommit call count")
	})

	t.Run("error rolls back", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var rollbackCount int
		mock.MockRollback = func() error {
			rollbackCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		txErr := errors.New("read-only tx failed")
		err := TransactionReadOnly(ctx, func(ctx context.Context) error {
			return txErr
		})
		require.ErrorIs(t, err, txErr)
		require.Equal(t, 1, rollbackCount, "MockRollback call count")
	})

	t.Run("ContextWithoutTransactions bypasses", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithoutTransactions(ctx)

		var called bool
		err := TransactionReadOnly(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})
}

func TestTransactionSavepoint_DB(t *testing.T) {
	t.Run("not in transaction uses regular transaction", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var commitCount int
		mock.MockCommit = func() error {
			commitCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := TransactionSavepoint(ctx, func(ctx context.Context) error {
			require.True(t, IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, commitCount, "MockCommit call count")
	})

	t.Run("in transaction uses savepoint", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1 // simulate active transaction
		var execCount int
		var execQueries []string
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			execQueries = append(execQueries, query)
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithSavepointFunc(ctx, func() string { return "test_sp" })

		var called bool
		err := TransactionSavepoint(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
		require.Equal(t, 2, execCount, "MockExec call count (savepoint + release)")
		require.Equal(t, "savepoint test_sp", execQueries[0])
		require.Equal(t, "release savepoint test_sp", execQueries[1])
	})

	t.Run("in transaction error rolls back to savepoint", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		mock.TxID = 1 // simulate active transaction
		var execCount int
		var execQueries []string
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			execQueries = append(execQueries, query)
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithSavepointFunc(ctx, func() string { return "test_sp" })

		txErr := errors.New("savepoint tx failed")
		err := TransactionSavepoint(ctx, func(ctx context.Context) error {
			return txErr
		})
		require.ErrorIs(t, err, txErr)
		require.Equal(t, 2, execCount, "MockExec call count (savepoint + rollback)")
		require.Equal(t, "savepoint test_sp", execQueries[0])
		require.Equal(t, "rollback to test_sp", execQueries[1])
	})

	t.Run("ContextWithoutTransactions bypasses", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)
		ctx = ContextWithoutTransactions(ctx)

		var called bool
		err := TransactionSavepoint(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})
}
