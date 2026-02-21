package sqldb

import (
	"context"
	"errors"
	"testing"
)

func TestUpdate(t *testing.T) {
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
		err := Update(t.Context(), ext, "users", Values{"name": "Bob"}, "id = $1", 42)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		wantQuery := "UPDATE users SET name=$2 WHERE id = $1"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{42, "Bob"})
	})

	t.Run("empty values error", func(t *testing.T) {
		_, ext := newTestConnExt()
		err := Update(t.Context(), ext, "users", Values{}, "id = $1", 42)
		if err == nil {
			t.Error("expected error for empty values")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		testErr := errors.New("update failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		err := Update(t.Context(), ext, "users", Values{"name": "Bob"}, "id = $1", 42)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestUpdateRowStruct(t *testing.T) {
	wantQuery := "UPDATE test_table SET name=$2, active=$3 WHERE id = $1"

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
		row := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		err := UpdateRowStruct(t.Context(), ext, "test_table", row)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(1), "Alice", true})
	})

	t.Run("with pointer", func(t *testing.T) {
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
		row := &reflectTestStruct{ID: 2, Name: "Bob", Active: false}
		err := UpdateRowStruct(t.Context(), ext, "test_table", row)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(2), "Bob", false})
	})

	t.Run("nil struct error", func(t *testing.T) {
		_, ext := newTestConnExt()
		err := UpdateRowStruct(t.Context(), ext, "test_table", (*reflectTestStruct)(nil))
		if err == nil {
			t.Error("expected error for nil struct")
		}
	})

	t.Run("non-struct error", func(t *testing.T) {
		_, ext := newTestConnExt()
		err := UpdateRowStruct(t.Context(), ext, "test_table", "not a struct")
		if err == nil {
			t.Error("expected error for non-struct")
		}
	})

	t.Run("no primary key error", func(t *testing.T) {
		_, ext := newTestConnExt()
		type noPK struct {
			Name string `db:"name"`
		}
		err := UpdateRowStruct(t.Context(), ext, "test_table", noPK{Name: "test"})
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		testErr := errors.New("update failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		row := reflectTestStruct{ID: 1, Name: "Alice"}
		err := UpdateRowStruct(t.Context(), ext, "test_table", row)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}
