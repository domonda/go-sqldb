package sqldb

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nilReflTestRow struct {
	TableName `db:"users"`

	ID   int    `db:"id,primarykey"`
	Name string `db:"name"`
}

// assertReturnsNilReflectorErr requires err to be non-nil and
// to mention the function name and a "nil StructReflector" hint,
// confirming the function returned an error rather than panicked.
func assertReturnsNilReflectorErr(t *testing.T, funcName string, err error) {
	t.Helper()
	require.Error(t, err, "%s with nil StructReflector should return an error", funcName)
	msg := err.Error()
	assert.Contains(t, msg, funcName, "error from %s should mention the function name", funcName)
	assert.Contains(t, msg, "nil StructReflector", "error from %s should mention nil StructReflector", funcName)
}

// ---------------------------------------------------------------------------
// Functions where nil refl is OK on the non-struct path
// ---------------------------------------------------------------------------

func TestNilReflector_NewRow_NonStruct(t *testing.T) {
	rows := NewMockRowsValue("id", int64(7))
	row := NewRow(rows, nil, NewQueryFormatter("$"), "SELECT id", nil)

	var got int64
	require.NoError(t, row.Scan(&got))
	assert.Equal(t, int64(7), got)
}

func TestNilReflector_NewRow_StructPath_ReturnsError(t *testing.T) {
	rows := NewMockRowsValue("id", int64(7)).WithRow(int64(8))
	row := NewRow(rows, nil, NewQueryFormatter("$"), "SELECT id", nil)

	var dest nilReflTestRow
	err := row.Scan(&dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil StructReflector")
}

func TestNilReflector_QueryRow_NonStruct(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"id"}, [][]driver.Value{{int64(42)}}, "SELECT id")

	var got int64
	err := QueryRow(t.Context(), conn, nil, conn, "SELECT id").Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, int64(42), got)
}

func TestNilReflector_QueryRowAs_NonStruct(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"n"}, [][]driver.Value{{int64(123)}}, "SELECT n")

	got, err := QueryRowAs[int64](t.Context(), conn, nil, conn, "SELECT n")
	require.NoError(t, err)
	assert.Equal(t, int64(123), got)
}

func TestNilReflector_QueryRowAs_StructPath_ReturnsError(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"id", "name"}, [][]driver.Value{{int64(1), "alice"}}, "SELECT * FROM t")

	_, err := QueryRowAs[nilReflTestRow](t.Context(), conn, nil, conn, "SELECT * FROM t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil StructReflector")
}

func TestNilReflector_QueryRowAsOr_NonStruct(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"n"}, [][]driver.Value{{int64(7)}}, "SELECT n")

	got, err := QueryRowAsOr[int64](t.Context(), conn, nil, conn, 99, "SELECT n")
	require.NoError(t, err)
	assert.Equal(t, int64(7), got)
}

func TestNilReflector_QueryRowAsOr_StructPath_ReturnsError(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"id", "name"}, [][]driver.Value{{int64(1), "alice"}}, "SELECT * FROM t")

	_, err := QueryRowAsOr(t.Context(), conn, nil, conn, nilReflTestRow{}, "SELECT * FROM t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil StructReflector")
}

func TestNilReflector_QueryRowAsStmt_NonStruct(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"n"}, [][]driver.Value{{int64(11)}}, "SELECT n")

	queryFunc, closeStmt, err := QueryRowAsStmt[int64](t.Context(), conn, nil, conn, "SELECT n")
	require.NoError(t, err)
	defer func() { require.NoError(t, closeStmt()) }()

	got, err := queryFunc(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(11), got)
}

func TestNilReflector_QueryRowsAsSlice_NonStruct(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"n"}, [][]driver.Value{{int64(1)}, {int64(2)}, {int64(3)}}, "SELECT n")

	got, err := QueryRowsAsSlice[int64](t.Context(), conn, nil, conn, UnlimitedMaxNumRows, "SELECT n")
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3}, got)
}

func TestNilReflector_QueryRowsAsSlice_StructPath_ReturnsError(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"id", "name"}, [][]driver.Value{{int64(1), "alice"}}, "SELECT * FROM t")

	_, err := QueryRowsAsSlice[nilReflTestRow](t.Context(), conn, nil, conn, UnlimitedMaxNumRows, "SELECT * FROM t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil StructReflector")
}

func TestNilReflector_QueryCallback_ScalarArgs(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"n"}, [][]driver.Value{{int64(10)}, {int64(20)}}, "SELECT n")

	var collected []int64
	err := QueryCallback(
		t.Context(), conn, nil, conn,
		func(n int64) error {
			collected = append(collected, n)
			return nil
		},
		"SELECT n",
	)
	require.NoError(t, err)
	assert.Equal(t, []int64{10, 20}, collected)
}

func TestNilReflector_QueryCallback_StructArg_ReturnsError(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$")).
		WithQueryResult([]string{"id", "name"}, [][]driver.Value{{int64(1), "alice"}}, "SELECT * FROM t")

	err := QueryCallback(
		t.Context(), conn, nil, conn,
		func(_ nilReflTestRow) error { return nil },
		"SELECT * FROM t",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil StructReflector")
}

func TestNilReflector_InsertReturning_NonStruct(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$"))
	conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
		return NewMockRowsValue("id", int64(99))
	}

	row := InsertReturning(
		t.Context(), conn, nil, StdReturningQueryBuilder{}, conn,
		"users", Values{"name": "alice"}, "id",
	)

	var got int64
	require.NoError(t, row.Scan(&got))
	assert.Equal(t, int64(99), got)
}

func TestNilReflector_UpdateReturningRow_NonStruct(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$"))
	conn.MockQuery = func(ctx context.Context, query string, args ...any) Rows {
		return NewMockRowsValue("id", int64(7))
	}

	row := UpdateReturningRow(
		t.Context(), conn, nil, StdReturningQueryBuilder{}, conn,
		"users", Values{"name": "alice"}, "id", "id = $1", 7,
	)

	var got int64
	require.NoError(t, row.Scan(&got))
	assert.Equal(t, int64(7), got)
}

// ---------------------------------------------------------------------------
// Functions where nil refl must always return an error (never panic)
// ---------------------------------------------------------------------------

func TestNilReflector_StructFunctions_AllReturnErrors(t *testing.T) {
	ctx := t.Context()
	conn := NewMockConn(NewQueryFormatter("$"))
	builder := testUpsertBuilder{}
	row := nilReflTestRow{ID: 1, Name: "alice"}
	rows := []nilReflTestRow{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}

	t.Run("QueryRowStruct", func(t *testing.T) {
		_, err := QueryRowStruct[nilReflTestRow](ctx, conn, nil, builder, conn, 1)
		assertReturnsNilReflectorErr(t, "QueryRowStruct", err)
	})

	t.Run("QueryRowStructOr", func(t *testing.T) {
		_, err := QueryRowStructOr(ctx, conn, nil, builder, conn, nilReflTestRow{}, 1)
		assertReturnsNilReflectorErr(t, "QueryRowStruct", err) // delegates to QueryRowStruct
	})

	t.Run("QueryStructCallback", func(t *testing.T) {
		err := QueryStructCallback(
			ctx, conn, nil, conn,
			func(_ nilReflTestRow) error { return nil },
			"SELECT * FROM users",
		)
		require.Error(t, err)
		// Wrapped with WrapErrorWithQuery, so just check the substring
		assert.True(t, strings.Contains(err.Error(), "nil StructReflector"),
			"expected 'nil StructReflector' in error, got: %s", err.Error())
	})

	t.Run("InsertRowStruct", func(t *testing.T) {
		err := InsertRowStruct(ctx, conn, nil, builder, conn, &row)
		assertReturnsNilReflectorErr(t, "InsertRowStruct", err)
	})

	t.Run("InsertRowStructStmt", func(t *testing.T) {
		_, _, err := InsertRowStructStmt[*nilReflTestRow](ctx, conn, nil, builder, conn)
		assertReturnsNilReflectorErr(t, "InsertRowStructStmt", err)
	})

	t.Run("InsertUniqueRowStruct", func(t *testing.T) {
		_, err := InsertUniqueRowStruct(ctx, conn, nil, builder, conn, &row, "id")
		assertReturnsNilReflectorErr(t, "InsertUniqueRowStruct", err)
	})

	t.Run("InsertRowStructs", func(t *testing.T) {
		err := InsertRowStructs(ctx, conn, nil, builder, conn, rows)
		assertReturnsNilReflectorErr(t, "InsertRowStructs", err)
	})

	t.Run("UpdateRowStruct", func(t *testing.T) {
		err := UpdateRowStruct(ctx, conn, nil, builder, conn, &row)
		assertReturnsNilReflectorErr(t, "UpdateRowStruct", err)
	})

	t.Run("UpdateRowStructStmt", func(t *testing.T) {
		_, _, err := UpdateRowStructStmt[*nilReflTestRow](ctx, conn, nil, builder, conn)
		assertReturnsNilReflectorErr(t, "UpdateRowStructStmt", err)
	})

	t.Run("UpdateRowStructs", func(t *testing.T) {
		err := UpdateRowStructs(ctx, conn, nil, builder, conn, rows)
		assertReturnsNilReflectorErr(t, "UpdateRowStructs", err)
	})

	t.Run("UpsertRowStruct", func(t *testing.T) {
		err := UpsertRowStruct(ctx, conn, nil, builder, conn, &row)
		assertReturnsNilReflectorErr(t, "UpsertRowStruct", err)
	})

	t.Run("UpsertRowStructStmt", func(t *testing.T) {
		_, _, err := UpsertRowStructStmt[*nilReflTestRow](ctx, conn, nil, builder, conn)
		assertReturnsNilReflectorErr(t, "UpsertRowStructStmt", err)
	})

	t.Run("UpsertRowStructs", func(t *testing.T) {
		err := UpsertRowStructs(ctx, conn, nil, builder, conn, rows)
		assertReturnsNilReflectorErr(t, "UpsertRowStructs", err)
	})

	t.Run("DeleteRowStruct", func(t *testing.T) {
		err := DeleteRowStruct(ctx, conn, nil, builder, conn, &row)
		assertReturnsNilReflectorErr(t, "DeleteRowStruct", err)
	})

	t.Run("DeleteRowStructStmt", func(t *testing.T) {
		_, _, err := DeleteRowStructStmt[*nilReflTestRow](ctx, conn, nil, builder, conn)
		assertReturnsNilReflectorErr(t, "DeleteRowStructStmt", err)
	})

	t.Run("DeleteRowStructs", func(t *testing.T) {
		err := DeleteRowStructs(ctx, conn, nil, builder, conn, rows)
		assertReturnsNilReflectorErr(t, "DeleteRowStructs", err)
	})
}
