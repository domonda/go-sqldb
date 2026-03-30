package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestQueryRow(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("id", "name").WithRow(int64(1), "Alice")
		}
		row := QueryRow(t.Context(), conn, refl, fmtr, "SELECT id, name FROM users WHERE id = $1", 1)
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			return NewMockRows("count").WithRow(int64(42))
		}
		val, err := QueryRowAs[int64](t.Context(), conn, refl, fmtr, "SELECT count(*) FROM users")
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("id")
		}
		_, err := QueryRowAs[int64](t.Context(), conn, refl, fmtr, "SELECT id FROM users WHERE id = $1", 999)
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("name").WithRow("Alice")
		}
		val, err := QueryRowAsOr(t.Context(), conn, refl, fmtr, "default", "SELECT name FROM users WHERE id = $1", 1)
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("name")
		}
		val, err := QueryRowAsOr(t.Context(), conn, refl, fmtr, "default", "SELECT name FROM users WHERE id = $1", 999)
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		testErr := errors.New("query failed")
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewErrRows(testErr)
		}
		_, err := QueryRowAsOr(t.Context(), conn, refl, fmtr, "default", "SELECT name FROM users")
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("name").WithRow("Alice")
		}
		queryFunc, closeStmt, err := QueryRowAsStmt[string](t.Context(), conn, refl, fmtr, "SELECT name FROM users WHERE id = $1")
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
		conn, refl, _, fmtr := newTestInterfaces()
		var prepareCount int
		prepErr := errors.New("prepare failed")
		conn.MockPrepare = func(ctx context.Context, query string) (Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		_, _, err := QueryRowAsStmt[string](t.Context(), conn, refl, fmtr, "SELECT name FROM users")
		if !errors.Is(err, prepErr) {
			t.Errorf("expected error wrapping %v, got: %v", prepErr, err)
		}
		if prepareCount != 1 {
			t.Errorf("MockPrepare called %d times, want 1", prepareCount)
		}
	})
}

func TestQueryRowByPrimaryKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("id", "name", "active").WithRow(int64(1), "Alice", true)
		}
		row, err := QueryRowByPrimaryKey[reflectTestStruct](t.Context(), conn, refl, builder, fmtr, int64(1))
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
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		// reflectTestStruct has 1 PK but we pass 2 values
		_, err := QueryRowByPrimaryKey[reflectTestStruct](t.Context(), conn, refl, builder, fmtr, 1, 2)
		if err == nil {
			t.Error("expected error for PK count mismatch")
		}
	})

	t.Run("no rows", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("id", "name", "active")
		}
		_, err := QueryRowByPrimaryKey[reflectTestStruct](t.Context(), conn, refl, builder, fmtr, int64(999))
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
		conn, _, _, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return NewMockRows("id", "name").WithRow(int64(1), "Alice")
		}
		m, err := QueryRowAsMap[string, any](t.Context(), conn, fmtr, "SELECT id, name FROM users WHERE id = $1", 1)
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
		conn, _, _, fmtr := newTestInterfaces()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("id")
		}
		_, err := QueryRowAsMap[string, any](t.Context(), conn, fmtr, "SELECT id FROM users WHERE id = $1", 999)
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			return NewMockRows("name").WithRow("Alice").WithRow("Bob").WithRow("Charlie")
		}
		names, err := QueryRowsAsSlice[string](t.Context(), conn, refl, fmtr, "SELECT name FROM users")
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			gotQuery = query
			return NewMockRows("id", "name", "active").
				WithRow(int64(1), "Alice", true).
				WithRow(int64(2), "Bob", false)
		}
		rows, err := QueryRowsAsSlice[reflectTestStruct](t.Context(), conn, refl, fmtr, "SELECT id, name, active FROM test_table")
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
		conn, refl, _, fmtr := newTestInterfaces()
		var queryCount int
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			queryCount++
			return NewMockRows("name")
		}
		names, err := QueryRowsAsSlice[string](t.Context(), conn, refl, fmtr, "SELECT name FROM users WHERE 1=0")
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

func TestQueryRowsAsStrings(t *testing.T) {
	t.Run("header plus data rows", func(t *testing.T) {
		// given
		conn, _, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("id", "name").
				WithRow(int64(1), "Alice").
				WithRow(int64(2), "Bob")
		}

		// when
		rows, err := QueryRowsAsStrings(t.Context(), conn, fmtr, "SELECT id, name FROM users")

		// then
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 3 {
			t.Fatalf("len = %d, want 3 (header + 2 data rows)", len(rows))
		}
		if rows[0][0] != "id" || rows[0][1] != "name" {
			t.Errorf("header row = %v, want [id name]", rows[0])
		}
		if rows[1][0] != "1" || rows[1][1] != "Alice" {
			t.Errorf("row[1] = %v, want [1 Alice]", rows[1])
		}
		if rows[2][0] != "2" || rows[2][1] != "Bob" {
			t.Errorf("row[2] = %v, want [2 Bob]", rows[2])
		}
	})

	t.Run("empty result returns only header", func(t *testing.T) {
		// given
		conn, _, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("col1", "col2")
		}

		// when
		rows, err := QueryRowsAsStrings(t.Context(), conn, fmtr, "SELECT col1, col2 FROM t WHERE 1=0")

		// then
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatalf("len = %d, want 1 (header only)", len(rows))
		}
		if rows[0][0] != "col1" || rows[0][1] != "col2" {
			t.Errorf("header row = %v, want [col1 col2]", rows[0])
		}
	})

	t.Run("query error propagates", func(t *testing.T) {
		// given
		conn, _, _, fmtr := newTestInterfaces()
		queryErr := errors.New("query failed")
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewErrRows(queryErr)
		}

		// when
		_, err := QueryRowsAsStrings(t.Context(), conn, fmtr, "SELECT 1")

		// then
		if !errors.Is(err, queryErr) {
			t.Errorf("expected error wrapping %v, got: %v", queryErr, err)
		}
	})
}

func TestQueryCallback(t *testing.T) {
	t.Run("scalar single column", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("name").
				WithRow("Alice").
				WithRow("Bob").
				WithRow("Charlie")
		}

		// when
		var names []string
		err := QueryCallback(t.Context(), conn, refl, fmtr,
			func(name string) { names = append(names, name) },
			"SELECT name FROM users",
		)

		// then
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"Alice", "Bob", "Charlie"}
		if len(names) != len(want) {
			t.Fatalf("len = %d, want %d", len(names), len(want))
		}
		for i, w := range want {
			if names[i] != w {
				t.Errorf("names[%d] = %q, want %q", i, names[i], w)
			}
		}
	})

	t.Run("callback returning error stops iteration", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("name").
				WithRow("Alice").
				WithRow("STOP").
				WithRow("Charlie")
		}
		stopErr := errors.New("stop")

		// when
		var names []string
		err := QueryCallback(t.Context(), conn, refl, fmtr,
			func(name string) error {
				if name == "STOP" {
					return stopErr
				}
				names = append(names, name)
				return nil
			},
			"SELECT name FROM users",
		)

		// then
		if !errors.Is(err, stopErr) {
			t.Errorf("expected error %v, got: %v", stopErr, err)
		}
		if len(names) != 1 || names[0] != "Alice" {
			t.Errorf("names = %v, want [Alice]", names)
		}
	})

	t.Run("callback with context arg", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("id").WithRow(int64(1)).WithRow(int64(2))
		}

		// when
		var ids []int64
		err := QueryCallback(t.Context(), conn, refl, fmtr,
			func(ctx context.Context, id int64) { ids = append(ids, id) },
			"SELECT id FROM items",
		)

		// then
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
			t.Errorf("ids = %v, want [1 2]", ids)
		}
	})

	t.Run("zero rows no error", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("name")
		}

		// when
		called := false
		err := QueryCallback(t.Context(), conn, refl, fmtr,
			func(name string) { called = true },
			"SELECT name FROM users WHERE 1=0",
		)

		// then
		if err != nil {
			t.Fatal(err)
		}
		if called {
			t.Error("callback should not be called for zero rows")
		}
	})

	t.Run("not a function returns error", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()

		// when
		err := QueryCallback(t.Context(), conn, refl, fmtr, "not a func", "SELECT 1")

		// then
		if err == nil {
			t.Error("expected error for non-function callback")
		}
	})

	t.Run("variadic function returns error", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()

		// when
		err := QueryCallback(t.Context(), conn, refl, fmtr, func(args ...string) {}, "SELECT 1")

		// then
		if err == nil {
			t.Error("expected error for variadic callback")
		}
	})

	t.Run("no arguments returns error", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()

		// when
		err := QueryCallback(t.Context(), conn, refl, fmtr, func() {}, "SELECT 1")

		// then
		if err == nil {
			t.Error("expected error for callback with no arguments")
		}
	})

	t.Run("column count mismatch returns error", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("name", "age").WithRow("Alice", int64(30))
		}

		// when – callback takes 1 arg but query returns 2 columns
		err := QueryCallback(t.Context(), conn, refl, fmtr,
			func(name string) {},
			"SELECT name, age FROM users",
		)

		// then
		if err == nil {
			t.Error("expected error for column count mismatch")
		}
	})

	t.Run("struct callback scans fields", func(t *testing.T) {
		// given
		conn, refl, _, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("id", "name", "active").
				WithRow(int64(1), "Alice", true).
				WithRow(int64(2), "Bob", false)
		}

		// when
		var results []reflectTestStruct
		err := QueryCallback(t.Context(), conn, refl, fmtr,
			func(row reflectTestStruct) { results = append(results, row) },
			"SELECT id, name, active FROM test_table",
		)

		// then
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Fatalf("len = %d, want 2", len(results))
		}
		if results[0].ID != 1 || results[0].Name != "Alice" || !results[0].Active {
			t.Errorf("results[0] = %+v, unexpected", results[0])
		}
		if results[1].ID != 2 || results[1].Name != "Bob" || results[1].Active {
			t.Errorf("results[1] = %+v, unexpected", results[1])
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
	conn := NewMockConn(NewQueryFormatter("$"))
	conn.MockQueryResults = map[string]Rows{
		"SELECT * FROM user": NewMockRows("id", "name", "email", "active").
			WithRow(int64(1), "Alice", "alice@example.com", true).
			WithRow(int64(2), "Bob", "bob@example.com", false).
			WithRow(int64(3), "Charlie", "charlie@example.com", true),
	}
	refl := NewTaggedStructReflector()
	fmtr := NewQueryFormatter("$")

	users, err := QueryRowsAsSlice[User](t.Context(), conn, refl, fmtr, "SELECT * FROM user")
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
