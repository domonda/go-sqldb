package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

type testUserRow struct {
	sqldb.TableName `db:"users"`
	ID              int64  `db:"id,primarykey"`
	Name            string `db:"name"`
	Active          bool   `db:"active"`
}

func TestQueryRow_DB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return sqldb.NewMockRows("id", "name").WithRow(int64(1), "Alice")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		row := QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", 1)
		var id int64
		var name string
		err := row.Scan(&id, &name)
		require.NoError(t, err)
		require.Equal(t, int64(1), id)
		require.Equal(t, "Alice", name)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, "SELECT id, name FROM users WHERE id = $1", gotQuery)
		require.Equal(t, []any{1}, gotArgs)
	})
}

func TestQueryRowAsStmt_DB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotArgs = args
			return sqldb.NewMockRows("name").WithRow("Alice")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		queryFunc, closeStmt, err := QueryRowAsStmt[string](ctx, "SELECT name FROM users WHERE id = $1")
		require.NoError(t, err)
		defer closeStmt()

		val, err := queryFunc(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, "Alice", val)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, []any{1}, gotArgs)
	})

	t.Run("prepare error", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var prepareCount int
		prepErr := errors.New("prepare failed")
		mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
			prepareCount++
			return nil, prepErr
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		_, _, err := QueryRowAsStmt[string](ctx, "SELECT name FROM users WHERE id = $1")
		require.ErrorIs(t, err, prepErr)
		require.Equal(t, 1, prepareCount, "MockPrepare call count")
	})
}

func TestQueryRowByPK_DB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return sqldb.NewMockRows("id", "name", "active").WithRow(int64(1), "Alice", true)
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		row, err := QueryRowByPK[testUserRow](ctx, int64(1))
		require.NoError(t, err)
		require.Equal(t, int64(1), row.ID)
		require.Equal(t, "Alice", row.Name)
		require.True(t, row.Active)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, "SELECT * FROM users WHERE id = $1", gotQuery)
		assertArgs(t, gotArgs, []any{int64(1)})
	})

	t.Run("no rows", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewMockRows("id", "name", "active")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		_, err := QueryRowByPK[testUserRow](ctx, int64(999))
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})
}

func TestQueryRowByPKOr_DB(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewMockRows("id", "name", "active").WithRow(int64(1), "Alice", true)
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		defaultVal := testUserRow{ID: 0, Name: "default"}
		row, err := QueryRowByPKOr(ctx, defaultVal, int64(1))
		require.NoError(t, err)
		require.Equal(t, int64(1), row.ID)
		require.Equal(t, "Alice", row.Name)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})

	t.Run("not found returns default", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewMockRows("id", "name", "active")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		defaultVal := testUserRow{ID: 0, Name: "default"}
		row, err := QueryRowByPKOr(ctx, defaultVal, int64(999))
		require.NoError(t, err)
		require.Equal(t, defaultVal, row)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})
}

func TestQueryRowAsMap_DB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotQuery string
		var gotArgs []any
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			gotArgs = args
			return sqldb.NewMockRows("id", "name").WithRow(int64(1), "Alice")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		m, err := QueryRowAsMap[string, any](ctx, "SELECT id, name FROM users WHERE id = $1", 1)
		require.NoError(t, err)
		require.Len(t, m, 2)
		require.Equal(t, int64(1), m["id"])
		require.Equal(t, "Alice", m["name"])
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, "SELECT id, name FROM users WHERE id = $1", gotQuery)
		require.Equal(t, []any{1}, gotArgs)
	})

	t.Run("no rows", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewMockRows("id")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		_, err := QueryRowAsMap[string, any](ctx, "SELECT id FROM users WHERE id = $1", 999)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})
}

func TestQueryRowsAsSlice_DB(t *testing.T) {
	t.Run("scalar values", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		var gotQuery string
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			gotQuery = query
			return sqldb.NewMockRows("name").WithRow("Alice").WithRow("Bob").WithRow("Charlie")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		names, err := QueryRowsAsSlice[string](ctx, "SELECT name FROM users")
		require.NoError(t, err)
		require.Equal(t, []string{"Alice", "Bob", "Charlie"}, names)
		require.Equal(t, 1, queryCount, "MockQuery call count")
		require.Equal(t, "SELECT name FROM users", gotQuery)
	})

	t.Run("empty result", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewMockRows("name")
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		names, err := QueryRowsAsSlice[string](ctx, "SELECT name FROM users WHERE 1=0")
		require.NoError(t, err)
		require.Nil(t, names)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})

	t.Run("struct values", func(t *testing.T) {
		mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
		var queryCount int
		mock.MockQuery = func(ctx context.Context, query string, args ...any) sqldb.Rows {
			queryCount++
			return sqldb.NewMockRows("id", "name", "active").
				WithRow(int64(1), "Alice", true).
				WithRow(int64(2), "Bob", false)
		}
		conn := mock.ConnExt()
		ctx := ContextWithConn(t.Context(), conn)

		rows, err := QueryRowsAsSlice[testUserRow](ctx, "SELECT id, name, active FROM users")
		require.NoError(t, err)
		require.Len(t, rows, 2)
		require.Equal(t, int64(1), rows[0].ID)
		require.Equal(t, "Alice", rows[0].Name)
		require.True(t, rows[0].Active)
		require.Equal(t, int64(2), rows[1].ID)
		require.Equal(t, "Bob", rows[1].Name)
		require.False(t, rows[1].Active)
		require.Equal(t, 1, queryCount, "MockQuery call count")
	})
}
