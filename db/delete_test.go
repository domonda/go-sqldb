package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

func TestDbDeleteRowStruct(t *testing.T) {
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

		err := db.DeleteRowStruct(ctx, UserRow{ID: 1, Name: "Alice", Active: true})
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "DELETE FROM users WHERE id = $1", gotQuery)
		assertArgs(t, gotArgs, []any{1})
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

		err := db.DeleteRowStruct(ctx, &UserRow{ID: 2, Name: "Bob", Active: false})
		require.NoError(t, err)
		require.Equal(t, 1, execCount, "MockExec call count")
		require.Equal(t, "DELETE FROM users WHERE id = $1", gotQuery)
		assertArgs(t, gotArgs, []any{2})
	})

	t.Run("no primary key error", func(t *testing.T) {
		type NoPKRow struct {
			db.TableName `db:"no_pk_table"`
			Name         string `db:"name"`
		}
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		ctx := testContext(t, mock)

		err := db.DeleteRowStruct(ctx, NoPKRow{Name: "test"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no mapped primary key")
	})

	t.Run("exec error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var execCount int
		testErr := errors.New("delete failed")
		mock.MockExec = func(ctx context.Context, query string, args ...any) error {
			execCount++
			return testErr
		}
		ctx := testContext(t, mock)

		err := db.DeleteRowStruct(ctx, UserRow{ID: 1, Name: "Alice"})
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, execCount, "MockExec call count")
	})
}
