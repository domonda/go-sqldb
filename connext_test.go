package sqldb

import (
	"context"
	"errors"
	"testing"
)

func TestTransactionExt(t *testing.T) {

	t.Run("error propagation", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var rolledBack bool
		conn.MockRollback = func() error {
			rolledBack = true
			return nil
		}
		txErr := errors.New("tx error")
		err := TransactionExt(t.Context(), ext, nil, func(tx ConnExt) error {
			return txErr
		})
		if !errors.Is(err, txErr) {
			t.Errorf("expected %v, got: %v", txErr, err)
		}
		if !rolledBack {
			t.Error("expected MockRollback to be called")
		}
	})
}

func TestTransactionResult(t *testing.T) {
	t.Run("success with result", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var committed bool
		conn.MockCommit = func() error {
			committed = true
			return nil
		}
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("count").WithRow(int64(42))
		}
		result, err := TransactionResult(t.Context(), ext, nil, func(tx ConnExt) (int64, error) {
			return QueryRowAs[int64](t.Context(), tx, tx, tx, "SELECT count(*) FROM users")
		})
		if err != nil {
			t.Fatal(err)
		}
		if result != 42 {
			t.Errorf("result = %d, want 42", result)
		}
		if !committed {
			t.Error("expected MockCommit to be called")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})

	t.Run("error returns zero result", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var rolledBack bool
		conn.MockRollback = func() error {
			rolledBack = true
			return nil
		}
		txErr := errors.New("tx error")
		result, err := TransactionResult(t.Context(), ext, nil, func(tx ConnExt) (string, error) {
			return "", txErr
		})
		if !errors.Is(err, txErr) {
			t.Errorf("expected %v, got: %v", txErr, err)
		}
		if result != "" {
			t.Errorf("result = %q, want empty", result)
		}
		if !rolledBack {
			t.Error("expected MockRollback to be called")
		}
	})
}
