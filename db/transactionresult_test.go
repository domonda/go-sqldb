package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestDebugNoTransaction(t *testing.T) {
	t.Run("executes function directly", func(t *testing.T) {
		called := false
		err := DebugNoTransaction(t.Context(), func(ctx context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates error", func(t *testing.T) {
		want := errors.New("test error")
		err := DebugNoTransaction(t.Context(), func(ctx context.Context) error {
			return want
		})
		assert.Equal(t, want, err)
	})
}

func TestDebugNoTransactionResult(t *testing.T) {
	t.Run("returns result", func(t *testing.T) {
		result, err := DebugNoTransactionResult(t.Context(), func(ctx context.Context) (int, error) {
			return 42, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("propagates error", func(t *testing.T) {
		want := errors.New("test error")
		_, err := DebugNoTransactionResult(t.Context(), func(ctx context.Context) (int, error) {
			return 0, want
		})
		assert.Equal(t, want, err)
	})
}

func TestTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result on success", func(t *testing.T) {
		result, err := TransactionResult(ctx, func(ctx context.Context) (string, error) {
			return "hello", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("returns error", func(t *testing.T) {
		want := errors.New("tx error")
		_, err := TransactionResult(ctx, func(ctx context.Context) (string, error) {
			return "", want
		})
		assert.ErrorContains(t, err, "tx error")
	})
}

func TestIsolatedTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result on success", func(t *testing.T) {
		result, err := IsolatedTransactionResult(ctx, func(ctx context.Context) (int, error) {
			return 99, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 99, result)
	})
}

func TestOptionalTransaction(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("with transaction", func(t *testing.T) {
		err := OptionalTransaction(ctx, true, func(ctx context.Context) error {
			assert.True(t, IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("without transaction", func(t *testing.T) {
		err := OptionalTransaction(ctx, false, func(ctx context.Context) error {
			return nil
		})
		require.NoError(t, err)
	})
}

func TestOptionalTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("with transaction returns result", func(t *testing.T) {
		result, err := OptionalTransactionResult(ctx, true, func(ctx context.Context) (string, error) {
			return "tx", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "tx", result)
	})

	t.Run("without transaction returns result", func(t *testing.T) {
		result, err := OptionalTransactionResult(ctx, false, func(ctx context.Context) (string, error) {
			return "no-tx", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "no-tx", result)
	})
}

func TestSerializedTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result", func(t *testing.T) {
		result, err := SerializedTransactionResult(ctx, func(ctx context.Context) (int, error) {
			return 7, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 7, result)
	})
}

func TestTransactionOptsResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result", func(t *testing.T) {
		result, err := TransactionOptsResult(ctx, nil, func(ctx context.Context) (string, error) {
			return "opts", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "opts", result)
	})
}

func TestTransactionReadOnly(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("executes within read-only transaction", func(t *testing.T) {
		err := TransactionReadOnly(ctx, func(ctx context.Context) error {
			assert.True(t, IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
	})
}

func TestTransactionReadOnlyResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result", func(t *testing.T) {
		result, err := TransactionReadOnlyResult(ctx, func(ctx context.Context) (int, error) {
			return 3, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 3, result)
	})
}

func TestTransactionSavepoint(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("without existing transaction uses Transaction", func(t *testing.T) {
		err := TransactionSavepoint(ctx, func(ctx context.Context) error {
			assert.True(t, IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("within transaction uses savepoint", func(t *testing.T) {
		err := Transaction(ctx, func(ctx context.Context) error {
			return TransactionSavepoint(ctx, func(ctx context.Context) error {
				return nil
			})
		})
		require.NoError(t, err)
	})

	t.Run("savepoint rollback on error", func(t *testing.T) {
		want := errors.New("sp error")
		err := Transaction(ctx, func(ctx context.Context) error {
			spErr := TransactionSavepoint(ctx, func(ctx context.Context) error {
				return want
			})
			assert.ErrorContains(t, spErr, "sp error")
			return nil // outer transaction succeeds
		})
		require.NoError(t, err)
	})
}

func TestTransactionSavepointResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result", func(t *testing.T) {
		result, err := TransactionSavepointResult(ctx, func(ctx context.Context) (string, error) {
			return "sp", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "sp", result)
	})
}

func TestContextWithGlobalConn(t *testing.T) {
	// given
	conn := new(sqldb.MockConn)
	saved := globalConn
	defer func() { globalConn = saved }()
	SetConn(conn)

	// when
	ctx := ContextWithGlobalConn(t.Context())

	// then
	assert.Equal(t, conn, Conn(ctx))
}

func TestSetQueryBuilder(t *testing.T) {
	// given
	saved := globalQueryBuilder
	defer func() { globalQueryBuilder = saved }()

	qb := sqldb.StdReturningQueryBuilder{}
	SetQueryBuilder(qb)

	// then
	assert.Equal(t, qb, QueryBuilder(t.Context()))
}

func TestSetStructReflector(t *testing.T) {
	// given
	saved := globalStructReflector
	defer func() { globalStructReflector = saved }()

	sr := sqldb.NewTaggedStructReflector()
	SetStructReflector(sr)

	// then
	assert.Equal(t, sr, StructReflector(t.Context()))
}
