package sqldb

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
)

func TestConnExt_WithConnection(t *testing.T) {
	conn1 := NewMockConn("$", nil, nil)
	reflector := NewTaggedStructReflector()
	formatter := NewQueryFormatter("$")
	builder := StdQueryBuilder{}
	ext := NewConnExt(conn1, reflector, formatter, builder)

	conn2 := NewMockConn("$", nil, nil)
	ext2 := ext.WithConnection(conn2)

	if ext2.Connection != conn2 {
		t.Error("expected new connection")
	}
	if ext2.StructReflector != reflector {
		t.Error("StructReflector not preserved")
	}
	if ext2.QueryFormatter != formatter {
		t.Error("QueryFormatter not preserved")
	}
	if ext2.QueryBuilder != builder {
		t.Error("QueryBuilder not preserved")
	}
}

func TestTransactionExt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var committed bool
		conn.MockCommit = func() error {
			committed = true
			return nil
		}
		var txExtReceived bool
		err := TransactionExt(t.Context(), ext, nil, func(tx *ConnExt) error {
			txExtReceived = true
			if tx.StructReflector != ext.StructReflector {
				t.Error("StructReflector not preserved in tx")
			}
			if tx.QueryFormatter != ext.QueryFormatter {
				t.Error("QueryFormatter not preserved in tx")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !txExtReceived {
			t.Error("txFunc was not called")
		}
		if !committed {
			t.Error("expected MockCommit to be called")
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var rolledBack bool
		conn.MockRollback = func() error {
			rolledBack = true
			return nil
		}
		txErr := errors.New("tx error")
		err := TransactionExt(t.Context(), ext, nil, func(tx *ConnExt) error {
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
			return NewMockRows([]string{"count"}, [][]driver.Value{{int64(42)}})
		}
		result, err := TransactionResult[int64](t.Context(), ext, nil, func(tx *ConnExt) (int64, error) {
			return QueryValue[int64](t.Context(), tx, "SELECT count(*) FROM users")
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
		result, err := TransactionResult[string](t.Context(), ext, nil, func(tx *ConnExt) (string, error) {
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
