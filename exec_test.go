package sqldb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// testUpsertBuilder implements [QueryBuilder], [UpsertQueryBuilder],
// and [ReturningQueryBuilder] for tests using PostgreSQL/SQLite-compatible
// ON CONFLICT syntax. This avoids importing driver packages from root tests.
type testUpsertBuilder struct {
	StdReturningQueryBuilder
}

func (b testUpsertBuilder) InsertUnique(formatter QueryFormatter, table string, columns []ColumnInfo, onConflict string) (query string, err error) {
	var q strings.Builder
	insert, err := b.Insert(formatter, table, columns)
	if err != nil {
		return "", err
	}
	q.WriteString(insert)
	if strings.HasPrefix(onConflict, "(") && strings.HasSuffix(onConflict, ")") {
		onConflict = onConflict[1 : len(onConflict)-1]
	}
	fmt.Fprintf(&q, " ON CONFLICT (%s) DO NOTHING", onConflict)
	return q.String(), nil
}

func (b testUpsertBuilder) Upsert(formatter QueryFormatter, table string, columns []ColumnInfo) (query string, err error) {
	hasNonPK := false
	for i := range columns {
		if !columns[i].PrimaryKey {
			hasNonPK = true
			break
		}
	}
	if !hasNonPK {
		return "", fmt.Errorf("Upsert requires at least one non-primary-key column")
	}
	var q strings.Builder
	insert, err := b.Insert(formatter, table, columns)
	if err != nil {
		return "", err
	}
	q.WriteString(insert)
	q.WriteString(` ON CONFLICT(`)
	first := true
	for i := range columns {
		if !columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			q.WriteByte(',')
		}
		columnName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		q.WriteString(columnName)
	}
	q.WriteString(`) DO UPDATE SET`)
	first = true
	for i := range columns {
		if columns[i].PrimaryKey {
			continue
		}
		if first {
			first = false
		} else {
			q.WriteByte(',')
		}
		columnName, err := formatter.FormatColumnName(columns[i].Name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&q, ` %s=%s`, columnName, formatter.FormatPlaceholder(i))
	}
	return q.String(), nil
}

// newTestInterfaces creates a MockConn and separate interface values for testing.
// Shared helper used across multiple test files.
// Returns a testUpsertBuilder which implements QueryBuilder, UpsertQueryBuilder,
// and ReturningQueryBuilder using PostgreSQL/SQLite ON CONFLICT syntax.
func newTestInterfaces() (conn *MockConn, refl StructReflector, builder testUpsertBuilder, fmtr QueryFormatter) {
	conn = NewMockConn(NewQueryFormatter("$"))
	return conn, NewTaggedStructReflector(), testUpsertBuilder{}, NewQueryFormatter("$")
}

func TestExec(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		var gotArgs []any
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		err := Exec(t.Context(), conn, fmtr, "DELETE FROM users WHERE id = $1", 42)
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
		conn, _, _, fmtr := newTestInterfaces()
		var execCount int
		testErr := errors.New("exec failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		err := Exec(t.Context(), conn, fmtr, "DELETE FROM users")
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestExecRowsAffected(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 3, nil
		}

		n, err := ExecRowsAffected(t.Context(), conn, fmtr, "UPDATE users SET active = $1", true)
		if err != nil {
			t.Fatal(err)
		}
		if n != 3 {
			t.Errorf("rows affected = %d, want 3", n)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		testErr := errors.New("exec failed")
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 0, testErr
		}

		n, err := ExecRowsAffected(t.Context(), conn, fmtr, "UPDATE users SET active = $1", true)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if n != 0 {
			t.Errorf("rows affected = %d, want 0", n)
		}
	})

	t.Run("nil mock returns context error", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		// MockExecRowsAffected is nil, should return ctx.Err()
		n, err := ExecRowsAffected(t.Context(), conn, fmtr, "UPDATE users SET active = $1", true)
		if err != nil {
			t.Errorf("expected nil error for non-cancelled context, got: %v", err)
		}
		if n != 0 {
			t.Errorf("rows affected = %d, want 0", n)
		}
	})

	t.Run("records exec", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 1, nil
		}

		_, err := ExecRowsAffected(t.Context(), conn, fmtr, "DELETE FROM users WHERE id = $1", 42)
		if err != nil {
			t.Fatal(err)
		}
		if len(conn.Recordings.Execs) != 1 {
			t.Fatalf("expected 1 recorded exec, got %d", len(conn.Recordings.Execs))
		}
		if conn.Recordings.Execs[0].Query != "DELETE FROM users WHERE id = $1" {
			t.Errorf("recorded query = %q, want %q", conn.Recordings.Execs[0].Query, "DELETE FROM users WHERE id = $1")
		}
	})

	t.Run("logs query", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		var buf strings.Builder
		conn.QueryLog = &buf
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 2, nil
		}

		_, err := ExecRowsAffected(t.Context(), conn, fmtr, "UPDATE users SET name = $1", "Alice")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "UPDATE users") {
			t.Errorf("expected query log to contain 'UPDATE users', got: %q", buf.String())
		}
	})
}

func TestExecStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		var gotArgs []any
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		execFunc, closeStmt, err := ExecStmt(t.Context(), conn, fmtr, "DELETE FROM users WHERE id = $1")
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
		conn, _, _, fmtr := newTestInterfaces()
		var prepareCount int
		prepErr := errors.New("prepare failed")
		conn.MockPrepare = func(ctx context.Context, query string) (Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		_, _, err := ExecStmt(t.Context(), conn, fmtr, "DELETE FROM users")
		if !errors.Is(err, prepErr) {
			t.Errorf("expected error wrapping %v, got: %v", prepErr, err)
		}
		if prepareCount != 1 {
			t.Errorf("MockPrepare called %d times, want 1", prepareCount)
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		var execCount int
		execErr := errors.New("exec failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return execErr
		}
		execFunc, closeStmt, err := ExecStmt(t.Context(), conn, fmtr, "DELETE FROM users WHERE id = $1")
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

func TestExecRowsAffectedStmt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		var execCount int
		var gotArgs []any
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			gotArgs = args
			return 7, nil
		}
		execFunc, closeStmt, err := ExecRowsAffectedStmt(t.Context(), conn, fmtr, "UPDATE users SET active = $1 WHERE role = $2")
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		n, err := execFunc(t.Context(), true, "admin")
		if err != nil {
			t.Fatal(err)
		}
		if n != 7 {
			t.Errorf("rows affected = %d, want 7", n)
		}
		if execCount != 1 {
			t.Errorf("MockExecRowsAffected called %d times, want 1", execCount)
		}
		assertArgs(t, gotArgs, []any{true, "admin"})
	})

	t.Run("prepare error", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		prepErr := errors.New("prepare failed")
		conn.MockPrepare = func(ctx context.Context, query string) (Stmt, error) {
			return nil, prepErr
		}
		_, _, err := ExecRowsAffectedStmt(t.Context(), conn, fmtr, "UPDATE users SET active = $1")
		if !errors.Is(err, prepErr) {
			t.Errorf("expected error wrapping %v, got: %v", prepErr, err)
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		execErr := errors.New("exec failed")
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 0, execErr
		}
		execFunc, closeStmt, err := ExecRowsAffectedStmt(t.Context(), conn, fmtr, "UPDATE users SET active = $1")
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		n, err := execFunc(t.Context(), true)
		if !errors.Is(err, execErr) {
			t.Errorf("expected error wrapping %v, got: %v", execErr, err)
		}
		if n != 0 {
			t.Errorf("rows affected = %d, want 0", n)
		}
	})

	t.Run("multiple calls", func(t *testing.T) {
		conn, _, _, fmtr := newTestInterfaces()
		var callCount int
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			callCount++
			return int64(callCount), nil
		}
		execFunc, closeStmt, err := ExecRowsAffectedStmt(t.Context(), conn, fmtr, "DELETE FROM users WHERE id = $1")
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		n1, err := execFunc(t.Context(), 1)
		if err != nil {
			t.Fatal(err)
		}
		n2, err := execFunc(t.Context(), 2)
		if err != nil {
			t.Fatal(err)
		}
		if n1 != 1 || n2 != 2 {
			t.Errorf("rows affected = (%d, %d), want (1, 2)", n1, n2)
		}
		if callCount != 2 {
			t.Errorf("MockExecRowsAffected called %d times, want 2", callCount)
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
