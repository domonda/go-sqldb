package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
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
		err := Update(t.Context(), conn, builder, fmtr, "users", Values{"name": "Bob"}, "id = $1", 42)
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
		conn, _, builder, fmtr := newTestInterfaces()
		_ = conn
		err := Update(t.Context(), conn, builder, fmtr, "users", Values{}, "id = $1", 42)
		if err == nil {
			t.Error("expected error for empty values")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, _, builder, fmtr := newTestInterfaces()
		var execCount int
		testErr := errors.New("update failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		err := Update(t.Context(), conn, builder, fmtr, "users", Values{"name": "Bob"}, "id = $1", 42)
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
		err := UpdateRowStruct(t.Context(), conn, refl, builder, fmtr, "test_table", row)
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
		err := UpdateRowStruct(t.Context(), conn, refl, builder, fmtr, "test_table", row)
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
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		err := UpdateRowStruct(t.Context(), conn, refl, builder, fmtr, "test_table", (*reflectTestStruct)(nil))
		if err == nil {
			t.Error("expected error for nil struct")
		}
	})

	t.Run("non-struct error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		err := UpdateRowStruct(t.Context(), conn, refl, builder, fmtr, "test_table", "not a struct")
		if err == nil {
			t.Error("expected error for non-struct")
		}
	})

	t.Run("no primary key error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		type noPK struct {
			Name string `db:"name"`
		}
		err := UpdateRowStruct(t.Context(), conn, refl, builder, fmtr, "test_table", noPK{Name: "test"})
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		testErr := errors.New("update failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		row := reflectTestStruct{ID: 1, Name: "Alice"}
		err := UpdateRowStruct(t.Context(), conn, refl, builder, fmtr, "test_table", row)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestUpdateReturningRow(t *testing.T) {
	t.Run("success scans returned row", func(t *testing.T) {
		// given
		conn, refl, builder, fmtr := newTestInterfaces()
		var gotQuery string
		var gotArgs []any
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			gotQuery = query
			gotArgs = args
			return NewMockRows("id", "name", "active").WithRow(int64(1), "Alice", true)
		}

		// when
		row := UpdateReturningRow(t.Context(), conn, refl, builder, fmtr,
			"test_table", Values{"name": "Alice"},
			"*", "id = $1", int64(1),
		)
		var dest reflectTestStruct
		err := row.Scan(&dest)

		// then
		require.NoError(t, err)
		assert.Equal(t, int64(1), dest.ID)
		assert.Equal(t, "Alice", dest.Name)
		assert.True(t, dest.Active)
		assert.Contains(t, gotQuery, "RETURNING")
		assertArgs(t, gotArgs, []any{int64(1), "Alice"})
	})

	t.Run("empty values returns error row", func(t *testing.T) {
		// given
		conn, refl, builder, fmtr := newTestInterfaces()

		// when
		row := UpdateReturningRow(t.Context(), conn, refl, builder, fmtr,
			"test_table", Values{},
			"*", "id = $1", int64(1),
		)
		var dest reflectTestStruct
		err := row.Scan(&dest)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no values passed")
	})

	t.Run("query error propagates", func(t *testing.T) {
		// given
		conn, refl, builder, fmtr := newTestInterfaces()
		queryErr := errors.New("query failed")
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewErrRows(queryErr)
		}

		// when
		row := UpdateReturningRow(t.Context(), conn, refl, builder, fmtr,
			"test_table", Values{"name": "Bob"},
			"*", "id = $1", int64(2),
		)
		var dest reflectTestStruct
		err := row.Scan(&dest)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, queryErr)
	})

	t.Run("no rows returns ErrNoRows", func(t *testing.T) {
		// given
		conn, refl, builder, fmtr := newTestInterfaces()
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewMockRows("id", "name", "active")
		}

		// when
		row := UpdateReturningRow(t.Context(), conn, refl, builder, fmtr,
			"test_table", Values{"name": "Nobody"},
			"*", "id = $1", int64(999),
		)
		var dest reflectTestStruct
		err := row.Scan(&dest)

		// then
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestUpdateReturningRows(t *testing.T) {
	t.Run("success returns rows", func(t *testing.T) {
		// given
		conn, _, builder, fmtr := newTestInterfaces()
		var gotQuery string
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			gotQuery = query
			return NewMockRows("id", "name", "active").
				WithRow(int64(1), "Alice", true).
				WithRow(int64(2), "Bob", false)
		}

		// when
		rows := UpdateReturningRows(t.Context(), conn, builder, fmtr,
			"test_table", Values{"active": false},
			"id, name, active", "active = $1", true,
		)
		defer rows.Close()

		// then
		assert.Contains(t, gotQuery, "RETURNING")
		cols, err := rows.Columns()
		require.NoError(t, err)
		assert.Equal(t, []string{"id", "name", "active"}, cols)

		var count int
		for rows.Next() {
			count++
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, 2, count)
	})

	t.Run("empty values returns error rows", func(t *testing.T) {
		// given
		conn, _, builder, fmtr := newTestInterfaces()

		// when
		rows := UpdateReturningRows(t.Context(), conn, builder, fmtr,
			"test_table", Values{},
			"id", "active = $1", true,
		)
		defer rows.Close()

		// then
		require.Error(t, rows.Err())
		assert.Contains(t, rows.Err().Error(), "no values passed")
	})

	t.Run("query error propagates via Err()", func(t *testing.T) {
		// given
		conn, _, builder, fmtr := newTestInterfaces()
		queryErr := errors.New("db down")
		conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
			return NewErrRows(queryErr)
		}

		// when
		rows := UpdateReturningRows(t.Context(), conn, builder, fmtr,
			"test_table", Values{"name": "X"},
			"id", "id = $1", int64(1),
		)
		defer rows.Close()

		// then
		require.ErrorIs(t, rows.Err(), queryErr)
	})
}
