package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestCheckTxOptionsCompatibility(t *testing.T) {
	type args struct {
		parent           *sql.TxOptions
		child            *sql.TxOptions
		defaultIsolation sql.IsolationLevel
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "nil, nil",
			args: args{
				parent: nil,
				child:  nil,
			},
		},
		{
			name: "nil, default",
			args: args{
				parent: nil,
				child:  &sql.TxOptions{},
			},
		},
		{
			name: "default, nil",
			args: args{
				parent: &sql.TxOptions{},
				child:  nil,
			},
		},
		{
			name: "default, default",
			args: args{
				parent: &sql.TxOptions{},
				child:  &sql.TxOptions{},
			},
		},
		{
			name: "nil, ReadOnly",
			args: args{
				parent: nil,
				child:  &sql.TxOptions{ReadOnly: true},
			},
		},
		{
			name: "ReadOnly, ReadOnly",
			args: args{
				parent: &sql.TxOptions{ReadOnly: true},
				child:  &sql.TxOptions{ReadOnly: true},
			},
			wantErr: false,
		},
		{
			name: "ReadOnly, nil",
			args: args{
				parent: &sql.TxOptions{ReadOnly: true},
				child:  nil,
			},
			wantErr: true,
		},
		{
			name: "ReadCommitted, ReadCommitted",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted},
			},
		},
		{
			name: "Serializable, ReadCommitted",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelSerializable},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted},
			},
		},
		{
			name: "ReadCommitted, Serializable",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted},
				child:  &sql.TxOptions{Isolation: sql.LevelSerializable},
			},
			wantErr: true,
		},
		{
			name: "ReadCommitted, Serializable/ReadOnly",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted},
				child:  &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true},
			},
			wantErr: true,
		},
		{
			name: "ReadCommitted/ReadOnly, ReadCommitted/ReadOnly",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true},
			},
		},
		{
			name: "Serializable/ReadOnly, ReadCommitted/ReadOnly",
			args: args{
				parent: &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true},
				child:  &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckTxOptionsCompatibility(tt.args.parent, tt.args.child, tt.args.defaultIsolation); (err != nil) != tt.wantErr {
				t.Errorf("CheckTxOptionsCompatibility() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNextTransactionID(t *testing.T) {
	// Always returns >= 1
	if NextTransactionID() < 1 {
		t.Fatal("NextTransactionID() < 1")
	}
}

func TestTransaction(t *testing.T) {
	t.Run("success commits", func(t *testing.T) {
		conn := NewMockConn("$", nil, nil)
		var commitCount int
		conn.MockCommit = func() error {
			commitCount++
			return nil
		}
		err := Transaction(t.Context(), conn, nil, func(tx Connection) error {
			if !tx.Transaction().Active() {
				t.Error("expected active transaction")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if commitCount != 1 {
			t.Errorf("MockCommit called %d times, want 1", commitCount)
		}
	})

	t.Run("error rolls back", func(t *testing.T) {
		conn := NewMockConn("$", nil, nil)
		var rollbackCount int
		conn.MockRollback = func() error {
			rollbackCount++
			return nil
		}
		txErr := errors.New("tx func failed")
		err := Transaction(t.Context(), conn, nil, func(tx Connection) error {
			return txErr
		})
		if !errors.Is(err, txErr) {
			t.Errorf("expected error wrapping %v, got: %v", txErr, err)
		}
		if rollbackCount != 1 {
			t.Errorf("MockRollback called %d times, want 1", rollbackCount)
		}
	})

	t.Run("already in transaction passes through", func(t *testing.T) {
		conn := NewMockConn("$", nil, nil)
		conn.TxID = 1 // simulate active transaction
		var called bool
		err := Transaction(t.Context(), conn, nil, func(tx Connection) error {
			called = true
			if tx != conn {
				t.Error("expected same connection to be passed through")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Error("expected txFunc to be called")
		}
	})
}

func TestIsolatedTransaction(t *testing.T) {
	t.Run("success commits", func(t *testing.T) {
		conn := NewMockConn("$", nil, nil)
		var commitCount int
		conn.MockCommit = func() error {
			commitCount++
			return nil
		}
		err := IsolatedTransaction(t.Context(), conn, nil, func(tx Connection) error {
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if commitCount != 1 {
			t.Errorf("MockCommit called %d times, want 1", commitCount)
		}
	})

	t.Run("error rolls back", func(t *testing.T) {
		conn := NewMockConn("$", nil, nil)
		var rollbackCount int
		conn.MockRollback = func() error {
			rollbackCount++
			return nil
		}
		txErr := errors.New("isolated tx failed")
		err := IsolatedTransaction(t.Context(), conn, nil, func(tx Connection) error {
			return txErr
		})
		if !errors.Is(err, txErr) {
			t.Errorf("expected %v, got: %v", txErr, err)
		}
		if rollbackCount != 1 {
			t.Errorf("MockRollback called %d times, want 1", rollbackCount)
		}
	})

	t.Run("panic rolls back and re-panics", func(t *testing.T) {
		conn := NewMockConn("$", nil, nil)
		var rollbackCount int
		conn.MockRollback = func() error {
			rollbackCount++
			return nil
		}
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic to be re-thrown")
			}
			if r != "test panic" {
				t.Errorf("unexpected panic value: %v", r)
			}
			if rollbackCount != 1 {
				t.Errorf("MockRollback called %d times, want 1", rollbackCount)
			}
		}()
		IsolatedTransaction(t.Context(), conn, nil, func(tx Connection) error {
			panic("test panic")
		})
	})

	t.Run("begin error", func(t *testing.T) {
		conn := NewMockConn("$", nil, nil)
		var beginCount int
		beginErr := errors.New("begin failed")
		conn.MockBegin = func(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error) {
			beginCount++
			return nil, beginErr
		}
		err := IsolatedTransaction(t.Context(), conn, nil, func(tx Connection) error {
			t.Error("txFunc should not be called on begin error")
			return nil
		})
		if !errors.Is(err, beginErr) {
			t.Errorf("expected error wrapping %v, got: %v", beginErr, err)
		}
		if beginCount != 1 {
			t.Errorf("MockBegin called %d times, want 1", beginCount)
		}
	})
}
