package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

func TestUpdate(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		err := db.Update(ctx, "users", sqldb.Values{"name": "Bob"}, "id = $1", 42)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "UPDATE users SET name=$2 WHERE id = $1", gotQuery)
		require.Equal(t, []any{42, "Bob"}, gotArgs)
	})

	t.Run("multiple values", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		err := db.Update(ctx, "users", sqldb.Values{"name": "Bob", "active": true}, "id = $1", 42)
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		// Values are sorted alphabetically: active, name
		require.Equal(t, "UPDATE users SET active=$2, name=$3 WHERE id = $1", gotQuery)
		require.Equal(t, []any{42, true, "Bob"}, gotArgs)
	})

	t.Run("empty values error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		ctx := testContext(t, mock)

		err := db.Update(ctx, "users", sqldb.Values{}, "id = $1", 42)
		require.Error(t, err)
	})

	t.Run("exec error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		testErr := errors.New("update failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		ctx := testContext(t, mock)

		err := db.Update(ctx, "users", sqldb.Values{"name": "Bob"}, "id = $1", 42)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}

func TestUpdateRowStruct(t *testing.T) {
	type UserRow struct {
		db.TableName `db:"users"`
		ID           int    `db:"id,primarykey"`
		Name         string `db:"name"`
		Active       bool   `db:"active"`
	}

	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		err := db.UpdateRowStruct(ctx, UserRow{ID: 1, Name: "Alice", Active: true})
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		// Columns in struct field order: id(PK), name, active
		// SET non-PK columns first, WHERE PK columns second
		require.Equal(t, "UPDATE users SET name=$1, active=$2 WHERE id = $3", gotQuery)
		require.Equal(t, []any{"Alice", true, 1}, gotArgs)
	})

	t.Run("with pointer", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		err := db.UpdateRowStruct(ctx, &UserRow{ID: 2, Name: "Bob", Active: false})
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "UPDATE users SET name=$1, active=$2 WHERE id = $3", gotQuery)
		require.Equal(t, []any{"Bob", false, 2}, gotArgs)
	})

	t.Run("with IgnoreColumns option", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		var gotQuery string
		var gotArgs []any
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			gotQuery = query
			gotArgs = args
			return nil
		}
		ctx := testContext(t, mock)

		err := db.UpdateRowStruct(ctx, UserRow{ID: 1, Name: "Alice", Active: true}, sqldb.IgnoreColumns("active"))
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "UPDATE users SET name=$1 WHERE id = $2", gotQuery)
		require.Equal(t, []any{"Alice", 1}, gotArgs)
	})

	t.Run("no primary key error", func(t *testing.T) {
		type NoPKRow struct {
			db.TableName `db:"users"`
			Name         string `db:"name"`
		}
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		ctx := testContext(t, mock)

		err := db.UpdateRowStruct(ctx, NoPKRow{Name: "test"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no mapped primary key")
	})

	t.Run("exec error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		testErr := errors.New("update failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		ctx := testContext(t, mock)

		err := db.UpdateRowStruct(ctx, UserRow{ID: 1, Name: "Alice"})
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}
