package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestDeleteRowStruct(t *testing.T) {
	wantQuery := "DELETE FROM test_table WHERE id = $1"

	t.Run("success", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		var gotArgs []any
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			gotQuery = query
			gotArgs = args
			return 1, nil
		}
		row := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		err := DeleteRowStruct(t.Context(), conn, refl, builder, fmtr, row)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExecRowsAffected called %d times, want 1", execCount)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(1)})
	})

	t.Run("with pointer", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var gotQuery string
		var gotArgs []any
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			gotQuery = query
			gotArgs = args
			return 1, nil
		}
		row := &reflectTestStruct{ID: 2, Name: "Bob", Active: false}
		err := DeleteRowStruct(t.Context(), conn, refl, builder, fmtr, row)
		if err != nil {
			t.Fatal(err)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(2)})
	})

	t.Run("no primary key error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		type noPKRow struct {
			TableName `db:"no_pk_table"`
			Name      string `db:"name"`
		}
		err := DeleteRowStruct(t.Context(), conn, refl, builder, fmtr, noPKRow{Name: "test"})
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		testErr := errors.New("delete failed")
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			return 0, testErr
		}
		row := reflectTestStruct{ID: 1, Name: "Alice"}
		err := DeleteRowStruct(t.Context(), conn, refl, builder, fmtr, row)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExecRowsAffected called %d times, want 1", execCount)
		}
	})

	t.Run("no rows affected", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 0, nil
		}
		row := reflectTestStruct{ID: 999, Name: "Ghost"}
		err := DeleteRowStruct(t.Context(), conn, refl, builder, fmtr, row)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
	})
}

func TestDeleteRowStructStmt(t *testing.T) {
	wantQuery := "DELETE FROM test_table WHERE id = $1"

	t.Run("success", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			gotQuery = query
			return 1, nil
		}
		deleteFunc, closeStmt, err := DeleteRowStructStmt[reflectTestStruct](t.Context(), conn, refl, builder, fmtr)
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		err = deleteFunc(t.Context(), reflectTestStruct{ID: 1, Name: "Alice", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		err = deleteFunc(t.Context(), reflectTestStruct{ID: 2, Name: "Bob", Active: false})
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 2 {
			t.Errorf("MockExecRowsAffected called %d times, want 2", execCount)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
	})

	t.Run("with pointer type param", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			gotQuery = query
			return 1, nil
		}
		deleteFunc, closeStmt, err := DeleteRowStructStmt[*reflectTestStruct](t.Context(), conn, refl, builder, fmtr)
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		err = deleteFunc(t.Context(), &reflectTestStruct{ID: 1, Name: "Alice", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExecRowsAffected called %d times, want 1", execCount)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
	})

	t.Run("no primary key error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		type noPKRow struct {
			TableName `db:"no_pk_table"`
			Name      string `db:"name"`
		}
		_, _, err := DeleteRowStructStmt[noPKRow](t.Context(), conn, refl, builder, fmtr)
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})

	t.Run("no rows affected", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 0, nil
		}
		deleteFunc, closeStmt, err := DeleteRowStructStmt[reflectTestStruct](t.Context(), conn, refl, builder, fmtr)
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		err = deleteFunc(t.Context(), reflectTestStruct{ID: 999, Name: "Ghost"})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
	})
}

func TestDeleteRowStructs(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		err := DeleteRowStructs[reflectTestStruct](t.Context(), conn, refl, builder, fmtr, nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("single item", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		var gotArgs []any
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			gotQuery = query
			gotArgs = args
			return 1, nil
		}
		items := []reflectTestStruct{{ID: 1, Name: "Alice", Active: true}}
		err := DeleteRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExecRowsAffected called %d times, want 1", execCount)
		}
		wantQuery := "DELETE FROM test_table WHERE id = $1"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(1)})
	})

	t.Run("multiple items uses transaction", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			gotQuery = query
			return 1, nil
		}
		items := []reflectTestStruct{
			{ID: 1, Name: "Alice", Active: true},
			{ID: 2, Name: "Bob", Active: false},
		}
		err := DeleteRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 2 {
			t.Errorf("MockExecRowsAffected called %d times, want 2", execCount)
		}
		wantQuery := "DELETE FROM test_table WHERE id = $1"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
	})

	t.Run("single pointer item", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var gotArgs []any
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			gotArgs = args
			return 1, nil
		}
		items := []*reflectTestStruct{{ID: 1, Name: "Alice", Active: true}}
		err := DeleteRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		assertArgs(t, gotArgs, []any{int64(1)})
	})

	t.Run("multiple pointer items", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			return 1, nil
		}
		items := []*reflectTestStruct{
			{ID: 1, Name: "Alice", Active: true},
			{ID: 2, Name: "Bob", Active: false},
		}
		err := DeleteRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 2 {
			t.Errorf("MockExecRowsAffected called %d times, want 2", execCount)
		}
	})

	t.Run("single no rows affected", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			return 0, nil
		}
		items := []reflectTestStruct{{ID: 999, Name: "Ghost"}}
		err := DeleteRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
	})

	t.Run("multiple second no rows affected", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		conn.MockExecRowsAffected = func(ctx context.Context, query string, args ...any) (int64, error) {
			execCount++
			if execCount == 1 {
				return 1, nil
			}
			return 0, nil
		}
		items := []reflectTestStruct{
			{ID: 1, Name: "Alice"},
			{ID: 999, Name: "Ghost"},
		}
		err := DeleteRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got: %v", err)
		}
	})
}
