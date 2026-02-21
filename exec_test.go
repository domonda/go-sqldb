package sqldb

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// newTestConnExt creates a MockConn and ConnExt for testing.
// Shared helper used across multiple test files.
func newTestConnExt() (*MockConn, *ConnExt) {
	conn := NewMockConn("$", nil, nil)
	return conn, NewConnExt(conn, NewTaggedStructReflector(), NewQueryFormatter("$"), StdQueryBuilder{})
}

func TestExec(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		var gotQuery string
		var gotArgs []any
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		err := Exec(t.Context(), ext, "DELETE FROM users WHERE id = $1", 42)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		if gotQuery != "DELETE FROM users WHERE id = $1" {
			t.Errorf("query = %q, want %q", gotQuery, "DELETE FROM users WHERE id = $1")
		}
		if len(gotArgs) != 1 || gotArgs[0] != 42 {
			t.Errorf("args = %v, want [42]", gotArgs)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		testErr := errors.New("exec failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		err := Exec(t.Context(), ext, "DELETE FROM users")
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestExecStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		var gotQuery string
		var gotArgs []any
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		execFunc, closeStmt, err := ExecStmt(t.Context(), ext, "DELETE FROM users WHERE id = $1")
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		err = execFunc(t.Context(), 42)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		if gotQuery != "DELETE FROM users WHERE id = $1" {
			t.Errorf("query = %q, want %q", gotQuery, "DELETE FROM users WHERE id = $1")
		}
		if len(gotArgs) != 1 || gotArgs[0] != 42 {
			t.Errorf("args = %v, want [42]", gotArgs)
		}
	})

	t.Run("prepare error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var prepareCount int
		prepErr := errors.New("prepare failed")
		conn.MockPrepare = func(ctx context.Context, query string) (Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		_, _, err := ExecStmt(t.Context(), ext, "DELETE FROM users")
		if !errors.Is(err, prepErr) {
			t.Errorf("expected error wrapping %v, got: %v", prepErr, err)
		}
		if prepareCount != 1 {
			t.Errorf("MockPrepare called %d times, want 1", prepareCount)
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		execErr := errors.New("exec failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return execErr
		}
		execFunc, closeStmt, err := ExecStmt(t.Context(), ext, "DELETE FROM users WHERE id = $1")
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		err = execFunc(t.Context(), 1)
		if !errors.Is(err, execErr) {
			t.Errorf("expected error wrapping %v, got: %v", execErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

// assertArgs is a test helper for comparing argument slices.
func assertArgs(t *testing.T, got, want []any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("args length = %d, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
		return
	}
	for i := range want {
		if fmt.Sprint(got[i]) != fmt.Sprint(want[i]) {
			t.Errorf("args[%d] = %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
		}
	}
}
