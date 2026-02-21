package sqldb

import (
	"context"
	"errors"
	"testing"
)

func TestUpsertStruct(t *testing.T) {
	wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, active=$3"

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
		err := UpsertStruct(t.Context(), ext, row)
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
		_, ext := newTestConnExt()
		type noPKRow struct {
			TableName `db:"no_pk_table"`
			Name      string `db:"name"`
		}
		err := UpsertStruct(t.Context(), ext, noPKRow{Name: "test"})
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		testErr := errors.New("upsert failed")
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		row := reflectTestStruct{ID: 1, Name: "Alice"}
		err := UpsertStruct(t.Context(), ext, row)
		if !errors.Is(err, testErr) {
			t.Errorf("expected error wrapping %v, got: %v", testErr, err)
		}
		if execCount != 1 {
			t.Errorf("MockExec called %d times, want 1", execCount)
		}
	})
}

func TestUpsertStructStmt(t *testing.T) {
	wantQuery := "INSERT INTO test_table(id,name,active) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=$2, active=$3"

	t.Run("success", func(t *testing.T) {
		conn, ext := newTestConnExt()
		var execCount int
		var gotQuery string
		conn.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			return nil
		}
		upsertFunc, doneFunc, err := UpsertStructStmt[reflectTestStruct](t.Context(), ext)
		if err != nil {
			t.Fatal(err)
		}
		defer doneFunc()

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

	t.Run("no primary key error", func(t *testing.T) {
		_, ext := newTestConnExt()
		type noPKRow struct {
			TableName `db:"no_pk_table"`
			Name      string `db:"name"`
		}
		_, _, err := UpsertStructStmt[noPKRow](t.Context(), ext)
		if err == nil {
			t.Error("expected error for struct without primary key")
		}
	})
}

func TestUpsertStructs(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		_, ext := newTestConnExt()
		err := UpsertStructs[reflectTestStruct](t.Context(), ext, nil)
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
		err := UpsertStructs(t.Context(), ext, items)
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
		}
		err := UpsertStructs(t.Context(), ext, items)
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
}
