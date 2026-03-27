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
	fmt.Fprintf(&q, " ON CONFLICT (%s) DO NOTHING RETURNING TRUE", onConflict)
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
