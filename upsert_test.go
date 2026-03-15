package sqldb

import (
	"context"
	"errors"
	"testing"
)

func TestUpsertRowStruct(t *testing.T) {
	wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, active=$3"

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
		err := UpsertRowStruct(t.Context(), conn, refl, builder, fmtr, row)
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

	t.Run("no primary key error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		_ = conn
		type noPKRow struct {
			TableName `db:"no_pk_table"`
			Name      string `db:"name"`
		}
		err := UpsertRowStruct(t.Context(), conn, refl, builder, fmtr, noPKRow{Name: "test"})
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})

	t.Run("with pointer", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var gotQuery string
		var gotArgs []any
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			gotQuery = query
			gotArgs = args
			return nil
		}
		row := &reflectTestStruct{ID: 2, Name: "Bob", Active: false}
		err := UpsertRowStruct(t.Context(), conn, refl, builder, fmtr, row)
		if err != nil {
			t.Fatal(err)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
		assertArgs(t, gotArgs, []any{int64(2), "Bob", false})
	})

	t.Run("exec error", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		testErr := errors.New("upsert failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		row := reflectTestStruct{ID: 1, Name: "Alice"}
		err := UpsertRowStruct(t.Context(), conn, refl, builder, fmtr, row)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestUpsertRowStructStmt(t *testing.T) {
	wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, active=$3"

	t.Run("success", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			return nil
		}
		upsertFunc, closeStmt, err := UpsertRowStructStmt[reflectTestStruct](t.Context(), conn, refl, builder, fmtr)
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		err = upsertFunc(t.Context(), reflectTestStruct{ID: 1, Name: "Alice", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		err = upsertFunc(t.Context(), reflectTestStruct{ID: 2, Name: "Bob", Active: false})
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 2 {
			t.Errorf("MockExec called %d times, want 2", execCount)
		}
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
	})

	t.Run("with pointer type param", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		var gotQuery string
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			return nil
		}
		upsertFunc, closeStmt, err := UpsertRowStructStmt[*reflectTestStruct](t.Context(), conn, refl, builder, fmtr)
		if err != nil {
			t.Fatal(err)
		}
		defer closeStmt()

		err = upsertFunc(t.Context(), &reflectTestStruct{ID: 1, Name: "Alice", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
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
		_, _, err := UpsertRowStructStmt[noPKRow](t.Context(), conn, refl, builder, fmtr)
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})
}

func TestUpsertRowStructs(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		err := UpsertRowStructs[reflectTestStruct](t.Context(), conn, refl, builder, fmtr, nil)
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
		err := UpsertRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
		wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, active=$3"
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
		}
		err := UpsertRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 2 {
			t.Errorf("MockExec called %d times, want 2", execCount)
		}
		wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, active=$3"
		if gotQuery != wantQuery {
			t.Errorf("query = %q, want %q", gotQuery, wantQuery)
		}
	})

	t.Run("single pointer item", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var gotArgs []any
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			gotArgs = args
			return nil
		}
		items := []*reflectTestStruct{{ID: 1, Name: "Alice", Active: true}}
		err := UpsertRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		assertArgs(t, gotArgs, []any{int64(1), "Alice", true})
	})

	t.Run("multiple pointer items", func(t *testing.T) {
		conn, refl, builder, fmtr := newTestInterfaces()
		var execCount int
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return nil
		}
		items := []*reflectTestStruct{
			{ID: 1, Name: "Alice", Active: true},
			{ID: 2, Name: "Bob", Active: false},
		}
		err := UpsertRowStructs(t.Context(), conn, refl, builder, fmtr, items)
		if err != nil {
			t.Fatal(err)
		}
		if execCount != 2 {
			t.Errorf("MockExec called %d times, want 2", execCount)
		}
	})
}
