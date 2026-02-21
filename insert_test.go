package sqldb

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
)

func TestInsert(t *testing.T) {
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
		err := Insert(t.Context(), ext, "users", Values{"name": "Alice", "age": 30})
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		wantQuery := "INSERT INTO users(age,name) VALUES($1,$2)"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{30, "Alice"})
	})

	t.Run("empty values error", func(t *testing.T) {
		_, ext := newTestConnExt()
		err := Insert(t.Context(), ext, "users", Values{})
		if err == nil {
			t.Error("expected error for empty values")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		testErr := errors.New("insert failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		err := Insert(t.Context(), ext, "users", Values{"name": "Alice"})
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestInsertUnique(t *testing.T) {
	t.Run("inserted", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows([]string{"true"}, [][]driver.Value{{true}})
		}
		inserted, err := InsertUnique(t.Context(), ext, "users", Values{"id": 1, "name": "Alice"}, "id")
		if err != nil {
			t.Fatal(err)
		}
		if !inserted {
			t.Error("expected inserted=true")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		wantQuery := "INSERT INTO users(id,name) VALUES($1,$2) ON CONFLICT (id) DO NOTHING RETURNING TRUE"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{1, "Alice"})
	})

	t.Run("conflict no insert", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows([]string{"true"}, nil)
		}
		inserted, err := InsertUnique(t.Context(), ext, "users", Values{"id": 1, "name": "Alice"}, "id")
		if err != nil {
			t.Fatal(err)
		}
		if inserted {
			t.Error("expected inserted=false")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})

	t.Run("empty values error", func(t *testing.T) {
		_, ext := newTestConnExt()
		_, err := InsertUnique(t.Context(), ext, "users", Values{}, "id")
		if err == nil {
			t.Error("expected error for empty values")
		}
	})
}

func TestInsertRowStruct(t *testing.T) {
	wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3)"

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
		err := InsertRowStruct(t.Context(), ext, row)
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
		err := InsertRowStruct(t.Context(), ext, row)
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

	t.Run("exec error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		testErr := errors.New("insert failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		row := reflectTestStruct{ID: 1, Name: "Alice"}
		err := InsertRowStruct(t.Context(), ext, row)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestInsertUniqueRowStruct(t *testing.T) {
	wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3) ON CONFLICT (id) DO NOTHING RETURNING TRUE"

	t.Run("inserted", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows([]string{"true"}, [][]driver.Value{{true}})
		}
		row := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		inserted, err := InsertUniqueRowStruct(t.Context(), ext, row, "id")
		if err != nil {
			t.Fatal(err)
		}
		if !inserted {
			t.Error("expected inserted=true")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(1), "Alice", true})
	})

	t.Run("conflict no insert", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows([]string{"true"}, nil)
		}
		row := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		inserted, err := InsertUniqueRowStruct(t.Context(), ext, row, "id")
		if err != nil {
			t.Fatal(err)
		}
		if inserted {
			t.Error("expected inserted=false")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}

func TestInsertRowStructStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		var gotQuery string
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			return nil
		}
		insertFunc, closeFunc, err := InsertRowStructStmt[reflectTestStruct](t.Context(), ext)
		if err != nil {
			t.Fatal(err)
		}
		defer closeFunc()

		err = insertFunc(t.Context(), reflectTestStruct{ID: 1, Name: "Alice", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		err = insertFunc(t.Context(), reflectTestStruct{ID: 2, Name: "Bob", Active: false})
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 2 {
			t.Errorf("MockExec called %d times, want 2", execCount)
		}
		wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3)"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
	})
}

func TestInsertRowStructs(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		_, ext := newTestConnExt()
		err := InsertRowStructs[reflectTestStruct](t.Context(), ext, nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("single item", func(t *testing.T) {
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
		items := []reflectTestStruct{{ID: 1, Name: "Alice", Active: true}}
		err := InsertRowStructs(t.Context(), ext, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3)"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(1), "Alice", true})
	})

	t.Run("multiple items uses transaction", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		var gotQuery string
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			return nil
		}
		items := []reflectTestStruct{
			{ID: 1, Name: "Alice", Active: true},
			{ID: 2, Name: "Bob", Active: false},
			{ID: 3, Name: "Charlie", Active: true},
		}
		err := InsertRowStructs(t.Context(), ext, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 3 {
			t.Errorf("MockExec called %d times, want 3", execCount)
		}
		wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3)"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
	})
}
