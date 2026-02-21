package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
)

func TestQueryRow(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows([]string{"id", "name"}, [][]driver.Value{{int64(1), "Alice"}})
		}
		row := QueryRow(t.Context(), ext, "SELECT id, name FROM users WHERE id = $1", 1)
		var id int64
		var name string
		if err := row.Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		if id != 1 {
			t.Errorf("id = %d, want 1", id)
		}
		if name != "Alice" {
			t.Errorf("name = %q, want %q", name, "Alice")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != "SELECT id, name FROM users WHERE id = $1" {
			t.Errorf("query = %q, want %q", gotQuery, "SELECT id, name FROM users WHERE id = $1")
		}
		assertArgs(t, gotArgs, []any{1})
	})
}

func TestQueryValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			return NewMockRows([]string{"count"}, [][]driver.Value{{int64(42)}})
		}
		val, err := QueryValue[int64](t.Context(), ext, "SELECT count(*) FROM users")
		if err != nil {
			t.Fatal(err)
		}
		if val != 42 {
			t.Errorf("val = %d, want 42", val)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != "SELECT count(*) FROM users" {
			t.Errorf("query = %q, want %q", gotQuery, "SELECT count(*) FROM users")
		}
	})

	t.Run("no rows error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows([]string{"id"}, nil)
		}
		_, err := QueryValue[int64](t.Context(), ext, "SELECT id FROM users WHERE id = $1", 999)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}

func TestQueryValueOr(t *testing.T) {
	t.Run("value found", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows([]string{"name"}, [][]driver.Value{{"Alice"}})
		}
		val, err := QueryValueOr(t.Context(), ext, "default", "SELECT name FROM users WHERE id = $1", 1)
		if err != nil {
			t.Fatal(err)
		}
		if val != "Alice" {
			t.Errorf("val = %q, want %q", val, "Alice")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != "SELECT name FROM users WHERE id = $1" {
			t.Errorf("query = %q, want %q", gotQuery, "SELECT name FROM users WHERE id = $1")
		}
		assertArgs(t, gotArgs, []any{1})
	})

	t.Run("no rows returns default", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows([]string{"name"}, nil)
		}
		val, err := QueryValueOr(t.Context(), ext, "default", "SELECT name FROM users WHERE id = $1", 999)
		if err != nil {
			t.Fatal(err)
		}
		if val != "default" {
			t.Errorf("val = %q, want %q", val, "default")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})

	t.Run("other error propagated", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		testErr := errors.New("query failed")
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewErrRows(testErr)
		}
		_, err := QueryValueOr(t.Context(), ext, "default", "SELECT name FROM users")
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}

func TestQueryValueStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows([]string{"name"}, [][]driver.Value{{"Alice"}})
		}
		queryFunc, closeStmt, err := QueryValueStmt[string](t.Context(), ext, "SELECT name FROM users WHERE id = $1")
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		val, err := queryFunc(t.Context(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if val != "Alice" {
			t.Errorf("val = %q, want %q", val, "Alice")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != "SELECT name FROM users WHERE id = $1" {
			t.Errorf("query = %q, want %q", gotQuery, "SELECT name FROM users WHERE id = $1")
		}
		assertArgs(t, gotArgs, []any{1})
	})

	t.Run("prepare error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var prepareCount int
		prepErr := errors.New("prepare failed")
		conn.MockPrepare = func(ctx context.Context, query string) (Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		_, _, err := QueryValueStmt[string](t.Context(), ext, "SELECT name FROM users")
		if !errors.Is(err, prepErr) {
			t.Errorf("expected error wrapping %v, got: %v", prepErr, err)
		}
		if prepareCount != 1 {
			t.Errorf("MockPrepare called %d times, want 1", prepareCount)
		}
	})
}

func TestQueryRowStructWithTableName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows(
				[]string{"id", "name", "active"},
				[][]driver.Value{{int64(1), "Alice", true}},
			)
		}
		row, err := QueryRowStructWithTableName[reflectTestStruct](t.Context(), ext, int64(1))
		if err != nil {
			t.Fatal(err)
		}
		if row.ID != 1 {
			t.Errorf("ID = %d, want 1", row.ID)
		}
		if row.Name != "Alice" {
			t.Errorf("Name = %q, want %q", row.Name, "Alice")
		}
		if !row.Active {
			t.Error("Active = false, want true")
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		wantQuery := "SELECT * FROM test_table WHERE id = $1"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(1)})
	})

	t.Run("pk count mismatch", func(t *testing.T) {
		_, ext := newTestConnExt()
		// reflectTestStruct has 1 PK but we pass 2 values
		_, err := QueryRowStructWithTableName[reflectTestStruct](t.Context(), ext, 1, 2)
		if err == nil {
			t.Error("expected error for PK count mismatch")
		}
	})

	t.Run("no rows", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows([]string{"id", "name", "active"}, nil)
		}
		_, err := QueryRowStructWithTableName[reflectTestStruct](t.Context(), ext, int64(999))
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}

func TestQueryRowAsMap(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows(
				[]string{"id", "name"},
				[][]driver.Value{{int64(1), "Alice"}},
			)
		}
		m, err := QueryRowAsMap[string, any](t.Context(), ext, "SELECT id, name FROM users WHERE id = $1", 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 2 {
			t.Fatalf("map length = %d, want 2", len(m))
		}
		if m["id"] != int64(1) {
			t.Errorf("m[id] = %v, want 1", m["id"])
		}
		if m["name"] != "Alice" {
			t.Errorf("m[name] = %v, want Alice", m["name"])
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != "SELECT id, name FROM users WHERE id = $1" {
			t.Errorf("query = %q, want %q", gotQuery, "SELECT id, name FROM users WHERE id = $1")
		}
		assertArgs(t, gotArgs, []any{1})
	})

	t.Run("no rows", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows([]string{"id"}, nil)
		}
		_, err := QueryRowAsMap[string, any](t.Context(), ext, "SELECT id FROM users WHERE id = $1", 999)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}

func TestQueryRowsAsSlice(t *testing.T) {
	t.Run("scalar values", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			return NewMockRows([]string{"name"}, [][]driver.Value{{"Alice"}, {"Bob"}, {"Charlie"}})
		}
		names, err := QueryRowsAsSlice[string](t.Context(), ext, "SELECT name FROM users")
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"Alice", "Bob", "Charlie"}
		if len(names) != len(want) {
			t.Fatalf("len = %d, want %d", len(names), len(want))
		}
		for i := range want {
			if names[i] != want[i] {
				t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
			}
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != "SELECT name FROM users" {
			t.Errorf("query = %q, want %q", gotQuery, "SELECT name FROM users")
		}
	})

	t.Run("struct values", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			return NewMockRows(
				[]string{"id", "name", "active"},
				[][]driver.Value{
					{int64(1), "Alice", true},
					{int64(2), "Bob", false},
				},
			)
		}
		rows, err := QueryRowsAsSlice[reflectTestStruct](t.Context(), ext, "SELECT id, name, active FROM test_table")
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("len = %d, want 2", len(rows))
		}
		if rows[0].ID != 1 || rows[0].Name != "Alice" || !rows[0].Active {
			t.Errorf("rows[0] = %+v, unexpected", rows[0])
		}
		if rows[1].ID != 2 || rows[1].Name != "Bob" || rows[1].Active {
			t.Errorf("rows[1] = %+v, unexpected", rows[1])
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
		if gotQuery != "SELECT id, name, active FROM test_table" {
			t.Errorf("query = %q, want %q", gotQuery, "SELECT id, name, active FROM test_table")
		}
	})

	t.Run("empty result", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows([]string{"name"}, nil)
		}
		names, err := QueryRowsAsSlice[string](t.Context(), ext, "SELECT name FROM users WHERE 1=0")
		if err != nil {
			t.Fatal(err)
		}
		if names != nil {
			t.Errorf("expected nil slice, got %v", names)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}
