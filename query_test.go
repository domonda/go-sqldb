package sqldb

import (
	"context"
	"database/sql"
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
			return NewMockRows("id", "name").WithRow(int64(1), "Alice")
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

func TestQueryRowAs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			return NewMockRows("count").WithRow(int64(42))
		}
		val, err := QueryRowAs[int64](t.Context(), ext, "SELECT count(*) FROM users")
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
			return NewMockRows("id")
		}
		_, err := QueryRowAs[int64](t.Context(), ext, "SELECT id FROM users WHERE id = $1", 999)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}

func TestQueryRowAsOr(t *testing.T) {
	t.Run("value found", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("name").WithRow("Alice")
		}
		val, err := QueryRowAsOr(t.Context(), ext, "default", "SELECT name FROM users WHERE id = $1", 1)
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
			return NewMockRows("name")
		}
		val, err := QueryRowAsOr(t.Context(), ext, "default", "SELECT name FROM users WHERE id = $1", 999)
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
		_, err := QueryRowAsOr(t.Context(), ext, "default", "SELECT name FROM users")
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if queryCount != 1 {
			t.Errorf("MockQuery called %d times, want 1", queryCount)
		}
	})
}

func TestQueryRowAsStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("name").WithRow("Alice")
		}
		queryFunc, closeStmt, err := QueryRowAsStmt[string](t.Context(), ext, "SELECT name FROM users WHERE id = $1")
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
		_, _, err := QueryRowAsStmt[string](t.Context(), ext, "SELECT name FROM users")
		if !errors.Is(err, prepErr) {
			t.Errorf("expected error wrapping %v, got: %v", prepErr, err)
		}
		if prepareCount != 1 {
			t.Errorf("MockPrepare called %d times, want 1", prepareCount)
		}
	})
}

func TestQueryRowByPK(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("id", "name", "active").WithRow(int64(1), "Alice", true)
		}
		row, err := QueryRowByPK[reflectTestStruct](t.Context(), ext, int64(1))
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
		_, err := QueryRowByPK[reflectTestStruct](t.Context(), ext, 1, 2)
		if err == nil {
			t.Error("expected error for PK count mismatch")
		}
	})

	t.Run("no rows", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("id", "name", "active")
		}
		_, err := QueryRowByPK[reflectTestStruct](t.Context(), ext, int64(999))
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
			return NewMockRows("id", "name").WithRow(int64(1), "Alice")
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
			return NewMockRows("id")
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
			return NewMockRows("name").WithRow("Alice").WithRow("Bob").WithRow("Charlie")
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
			return NewMockRows("id", "name", "active").
				WithRow(int64(1), "Alice", true).
				WithRow(int64(2), "Bob", false)
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
			return NewMockRows("name")
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

type User struct {
	TableName `db:"user"`

	ID     int64  `db:"id,primarykey"`
	Name   string `db:"name"`
	Email  string `db:"email"`
	Active bool   `db:"active"`
}

func TestQueryRowsAsSlice_MockQueryResults(t *testing.T) {
	conn := NewMockConn("$", nil, nil)
	conn.MockQueryResults = map[string]Rows{
		"SELECT * FROM user": NewMockRows("id", "name", "email", "active").
			WithRow(int64(1), "Alice", "alice@example.com", true).
			WithRow(int64(2), "Bob", "bob@example.com", false).
			WithRow(int64(3), "Charlie", "charlie@example.com", true),
	}
	ext := NewConnExt(conn, NewTaggedStructReflector(), NewQueryFormatter("$"), StdQueryBuilder{})

	users, err := QueryRowsAsSlice[User](t.Context(), ext, "SELECT * FROM user")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Fatalf("len = %d, want 3", len(users))
	}

	want := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Active: true},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Active: false},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", Active: true},
	}
	for i, w := range want {
		if users[i].ID != w.ID {
			t.Errorf("users[%d].ID = %d, want %d", i, users[i].ID, w.ID)
		}
		if users[i].Name != w.Name {
			t.Errorf("users[%d].Name = %q, want %q", i, users[i].Name, w.Name)
		}
		if users[i].Email != w.Email {
			t.Errorf("users[%d].Email = %q, want %q", i, users[i].Email, w.Email)
		}
		if users[i].Active != w.Active {
			t.Errorf("users[%d].Active = %v, want %v", i, users[i].Active, w.Active)
		}
	}
}
