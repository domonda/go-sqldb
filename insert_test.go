package sqldb

import (
	"context"
	"errors"
	"testing"
)

func TestInsert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, _, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		var gotArgs []any
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		err := Insert(t.Context(), conn, builder, fmtr, "users", Values{"name": "Alice", "age": 30})
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
		conn, _, builder, fmtr := newTestInterfaces()
		_ = conn
		err := Insert(t.Context(), conn, builder, fmtr, "users", Values{})
		if err == nil {
			t.Error("expected error for empty values")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, _, builder, fmtr := newTestInterfaces()
		var execCount int
		testErr := errors.New("insert failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		err := Insert(t.Context(), conn, builder, fmtr, "users", Values{"name": "Alice"})
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestInsertReturning(t *testing.T) {
	t.Run("scan single value", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			gotQuery = query
			gotArgs = args
			return NewMockRows("id").WithRow(int64(42))
		}
		var id int64
		err := InsertReturning(t.Context(), conn, refl, builder, fmtr, "users", Values{"name": "Alice", "age": 30}, "id").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		if id != 42 {
			t.Errorf("id = %d, want 42", id)
		}
		wantQuery := "INSERT INTO users(age,name) VALUES($1,$2) RETURNING id"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{30, "Alice"})
	})

	t.Run("scan multiple values", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("id", "created_at").WithRow(int64(1), "2025-01-01T00:00:00Z")
		}
		var id int64
		var createdAt string
		err := InsertReturning(t.Context(), conn, refl, builder, fmtr, "users", Values{"name": "Bob"}, "id, created_at").Scan(&id, &createdAt)
		if err != nil {
			t.Fatal(err)
		}
		if id != 1 {
			t.Errorf("id = %d, want 1", id)
		}
		if createdAt != "2025-01-01T00:00:00Z" {
			t.Errorf("createdAt = %q, want %q", createdAt, "2025-01-01T00:00:00Z")
		}
	})

	t.Run("empty values error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		var id int64
		err := InsertReturning(t.Context(), conn, refl, builder, fmtr, "users", Values{}, "id").Scan(&id)
		if err == nil {
			t.Error("expected error for empty values")
		}
	})

	t.Run("query error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		queryErr := errors.New("insert failed")
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewErrRows(queryErr)
		}
		var id int64
		err := InsertReturning(t.Context(), conn, refl, builder, fmtr, "users", Values{"name": "Alice"}, "id").Scan(&id)
		if !errors.Is(err, queryErr) {
			t.Errorf("expected error wrapping %v, got: %v", queryErr, err)
		}
	})
}

func TestInsertUnique(t *testing.T) {
	t.Run("inserted", func(t *testing.T) {
		conn, _, builder, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("true").WithRow(true)
		}
		inserted, err := InsertUnique(t.Context(), conn, builder, fmtr, "users", Values{"id": 1, "name": "Alice"}, "id")
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
		conn, _, builder, fmtr := newTestInterfaces()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("true")
		}
		inserted, err := InsertUnique(t.Context(), conn, builder, fmtr, "users", Values{"id": 1, "name": "Alice"}, "id")
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
		conn, _, builder, fmtr := newTestInterfaces()
		_ = conn
		_, err := InsertUnique(t.Context(), conn, builder, fmtr, "users", Values{}, "id")
		if err == nil {
			t.Error("expected error for empty values")
		}
	})
}

func TestInsertRowStruct(t *testing.T) {
	wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3)"

	t.Run("success", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
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
		err := InsertRowStruct(t.Context(), conn, refl, builder, fmtr, row)
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
		conn, refl, builder, fmtr := newTestInterfaces()
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
		err := InsertRowStruct(t.Context(), conn, refl, builder, fmtr, row)
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
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		testErr := errors.New("insert failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		row := reflectTestStruct{ID: 1, Name: "Alice"}
		err := InsertRowStruct(t.Context(), conn, refl, builder, fmtr, row)
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
		conn, refl, builder, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("true").WithRow(true)
		}
		row := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		inserted, err := InsertUniqueRowStruct(t.Context(), conn, refl, builder, fmtr, row, "id")
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
		conn, refl, builder, fmtr := newTestInterfaces()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("true")
		}
		row := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		inserted, err := InsertUniqueRowStruct(t.Context(), conn, refl, builder, fmtr, row, "id")
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
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			return nil
		}
		insertFunc, closeStmt, err := InsertRowStructStmt[reflectTestStruct](t.Context(), conn, refl, builder, fmtr)
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

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
		conn, refl, builder, fmtr := newTestInterfaces()
		err := InsertRowStructs[reflectTestStruct](t.Context(), conn, refl, builder, fmtr, nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("single item", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
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
		err := InsertRowStructs(t.Context(), conn, refl, builder, fmtr, items)
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
		conn, refl, builder, fmtr := newTestInterfaces()
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
		err := InsertRowStructs(t.Context(), conn, refl, builder, fmtr, items)
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
