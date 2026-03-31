package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

func TestDebugNoTransaction(t *testing.T) {
	t.Run("executes function directly", func(t *testing.T) {
		called := false
		err := db.DebugNoTransaction(t.Context(), func(ctx context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates error", func(t *testing.T) {
		want := errors.New("test error")
		err := db.DebugNoTransaction(t.Context(), func(ctx context.Context) error {
			return want
		})
		assert.Equal(t, want, err)
	})
}

func TestDebugNoTransactionResult(t *testing.T) {
	t.Run("returns result", func(t *testing.T) {
		result, err := db.DebugNoTransactionResult(t.Context(), func(ctx context.Context) (int, error) {
			return 42, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("propagates error", func(t *testing.T) {
		want := errors.New("test error")
		_, err := db.DebugNoTransactionResult(t.Context(), func(ctx context.Context) (int, error) {
			return 0, want
		})
		assert.Equal(t, want, err)
	})
}

func TestTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result on success", func(t *testing.T) {
		result, err := db.TransactionResult(ctx, func(ctx context.Context) (string, error) {
			return "hello", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("returns error", func(t *testing.T) {
		want := errors.New("tx error")
		_, err := db.TransactionResult(ctx, func(ctx context.Context) (string, error) {
			return "", want
		})
		assert.ErrorContains(t, err, "tx error")
	})
}

func TestIsolatedTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result on success", func(t *testing.T) {
		result, err := db.IsolatedTransactionResult(ctx, func(ctx context.Context) (int, error) {
			return 99, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 99, result)
	})
}

func TestOptionalTransaction(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("with transaction", func(t *testing.T) {
		err := db.OptionalTransaction(ctx, true, func(ctx context.Context) error {
			assert.True(t, db.IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("without transaction", func(t *testing.T) {
		err := db.OptionalTransaction(ctx, false, func(ctx context.Context) error {
			return nil
		})
		require.NoError(t, err)
	})
}

func TestOptionalTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("with transaction returns result", func(t *testing.T) {
		result, err := db.OptionalTransactionResult(ctx, true, func(ctx context.Context) (string, error) {
			return "tx", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "tx", result)
	})

	t.Run("without transaction returns result", func(t *testing.T) {
		result, err := db.OptionalTransactionResult(ctx, false, func(ctx context.Context) (string, error) {
			return "no-tx", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "no-tx", result)
	})
}

func TestSerializedTransactionResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result", func(t *testing.T) {
		result, err := db.SerializedTransactionResult(ctx, func(ctx context.Context) (int, error) {
			return 7, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 7, result)
	})
}

func TestTransactionOptsResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result", func(t *testing.T) {
		result, err := db.TransactionOptsResult(ctx, nil, func(ctx context.Context) (string, error) {
			return "opts", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "opts", result)
	})
}

func TestTransactionReadOnly(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("executes within read-only transaction", func(t *testing.T) {
		err := db.TransactionReadOnly(ctx, func(ctx context.Context) error {
			assert.True(t, db.IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
	})
}

func TestTransactionReadOnlyResult(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("returns result", func(t *testing.T) {
		result, err := db.TransactionReadOnlyResult(ctx, func(ctx context.Context) (int, error) {
			return 3, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 3, result)
	})
}

func TestTransactionSavepoint(t *testing.T) {
	ctx := testContext(t, new(sqldb.MockConn))

	t.Run("without existing transaction uses Transaction", func(t *testing.T) {
		err := db.TransactionSavepoint(ctx, func(ctx context.Context) error {
			assert.True(t, db.IsTransaction(ctx))
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("within transaction uses savepoint", func(t *testing.T) {
		err := db.Transaction(ctx, func(ctx context.Context) error {
			return db.TransactionSavepoint(ctx, func(ctx context.Context) error {
				return nil
			})
		})
		require.NoError(t, err)
	})

	t.Run("savepoint rollback on error", func(t *testing.T) {
		want := errors.New("sp error")
		err := db.Transaction(ctx, func(ctx context.Context) error {
			spErr := db.TransactionSavepoint(ctx, func(ctx context.Context) error {
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
		result, err := db.TransactionSavepointResult(ctx, func(ctx context.Context) (string, error) {
			return "sp", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "sp", result)
	})
}

func TestContextWithGlobalConn(t *testing.T) {
	conn := new(sqldb.MockConn)
	db.SetConn(conn)
	t.Cleanup(func() { db.SetConn(sqldb.NewErrConn(sqldb.ErrNoDatabaseConnection)) })

	ctx := db.ContextWithGlobalConn(t.Context())
	assert.Equal(t, conn, db.Conn(ctx))
}

func TestSetQueryBuilder(t *testing.T) {
	qb := sqldb.StdReturningQueryBuilder{}
	db.SetQueryBuilder(qb)
	t.Cleanup(func() { db.SetQueryBuilder(sqldb.StdReturningQueryBuilder{}) })

	assert.Equal(t, qb, db.QueryBuilder(t.Context()))
}

func TestSetStructReflector(t *testing.T) {
	sr := sqldb.NewTaggedStructReflector()
	db.SetStructReflector(sr)
	t.Cleanup(func() { db.SetStructReflector(sqldb.NewTaggedStructReflector()) })

	assert.Equal(t, sr, db.StructReflector(t.Context()))
}
