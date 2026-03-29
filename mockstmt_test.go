package sqldb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockStmt_PreparedQuery(t *testing.T) {
	stmt := &MockStmt{Prepared: "SELECT $1"}
	assert.Equal(t, "SELECT $1", stmt.PreparedQuery())
}

func TestMockStmt_Exec(t *testing.T) {
	t.Run("nil mock returns context error", func(t *testing.T) {
		stmt := &MockStmt{}
		err := stmt.Exec(t.Context())
		assert.NoError(t, err)
	})

	t.Run("canceled context", func(t *testing.T) {
		stmt := &MockStmt{}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := stmt.Exec(ctx)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("custom mock", func(t *testing.T) {
		called := false
		stmt := &MockStmt{
			MockExec: func(ctx context.Context, args ...any) error {
				called = true
				return nil
			},
		}
		require.NoError(t, stmt.Exec(t.Context()))
		assert.True(t, called)
	})
}

func TestMockStmt_ExecRowsAffected(t *testing.T) {
	t.Run("nil mock returns zero", func(t *testing.T) {
		stmt := &MockStmt{}
		n, err := stmt.ExecRowsAffected(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
	})

	t.Run("custom mock", func(t *testing.T) {
		stmt := &MockStmt{
			MockExecRowsAffected: func(ctx context.Context, args ...any) (int64, error) {
				return 5, nil
			},
		}
		n, err := stmt.ExecRowsAffected(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int64(5), n)
	})
}

func TestMockStmt_Query(t *testing.T) {
	t.Run("nil mock returns err rows", func(t *testing.T) {
		stmt := &MockStmt{}
		rows := stmt.Query(t.Context())
		defer rows.Close()
		assert.False(t, rows.Next())
	})
}

func TestMockStmt_Close(t *testing.T) {
	t.Run("nil mock returns nil", func(t *testing.T) {
		stmt := &MockStmt{}
		assert.NoError(t, stmt.Close())
	})

	t.Run("custom mock", func(t *testing.T) {
		closed := false
		stmt := &MockStmt{
			MockClose: func() error {
				closed = true
				return nil
			},
		}
		require.NoError(t, stmt.Close())
		assert.True(t, closed)
	})
}
