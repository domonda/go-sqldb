package conntest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runPrepareTests(t *testing.T, config Config) {
	t.Run("PreparedQuery", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		query := fmt.Sprintf( /*sql*/ `INSERT INTO conntest_simple (id, val) VALUES (%s, %s)`, conn.FormatPlaceholder(0), conn.FormatPlaceholder(1))

		// when
		stmt, err := conn.Prepare(ctx, query)
		require.NoError(t, err)
		defer stmt.Close() //nolint:errcheck

		// then
		assert.Equal(t, query, stmt.PreparedQuery())
	})

	t.Run("Exec", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		query := fmt.Sprintf( /*sql*/ `INSERT INTO conntest_simple (id, val) VALUES (%s, %s)`, conn.FormatPlaceholder(0), conn.FormatPlaceholder(1))

		stmt, err := conn.Prepare(ctx, query)
		require.NoError(t, err)
		defer stmt.Close() //nolint:errcheck

		// when
		err = stmt.Exec(ctx, 1, "prepared-value")

		// then
		require.NoError(t, err)
		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "prepared-value", got.Val)
	})

	t.Run("ExecRowsAffected", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		require.NoError(t, sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, []simpleRow{
			{ID: 1, Val: "a"},
			{ID: 2, Val: "b"},
		}))

		query := /*sql*/ `DELETE FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		stmt, err := conn.Prepare(ctx, query)
		require.NoError(t, err)
		defer stmt.Close() //nolint:errcheck

		// when
		n, err := stmt.ExecRowsAffected(ctx, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)
	})

	t.Run("Query", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "queried"})

		query := /*sql*/ `SELECT val FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		stmt, err := conn.Prepare(ctx, query)
		require.NoError(t, err)
		defer stmt.Close() //nolint:errcheck

		// when
		rows := stmt.Query(ctx, 1)
		require.True(t, rows.Next())
		var val string
		require.NoError(t, rows.Scan(&val))
		require.NoError(t, rows.Close())

		// then
		assert.Equal(t, "queried", val)
	})

	t.Run("MultipleExec", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		query := fmt.Sprintf( /*sql*/ `INSERT INTO conntest_simple (id, val) VALUES (%s, %s)`, conn.FormatPlaceholder(0), conn.FormatPlaceholder(1))
		stmt, err := conn.Prepare(ctx, query)
		require.NoError(t, err)
		defer stmt.Close() //nolint:errcheck

		// when
		require.NoError(t, stmt.Exec(ctx, 1, "first"))
		require.NoError(t, stmt.Exec(ctx, 2, "second"))
		require.NoError(t, stmt.Exec(ctx, 3, "third"))

		// then
		assert.Equal(t, "first", querySimpleRow(t, conn, qb, 1).Val)
		assert.Equal(t, "second", querySimpleRow(t, conn, qb, 2).Val)
		assert.Equal(t, "third", querySimpleRow(t, conn, qb, 3).Val)
	})

	t.Run("Close", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		query := /*sql*/ `SELECT val FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		stmt, err := conn.Prepare(ctx, query)
		require.NoError(t, err)

		// when
		err = stmt.Close()

		// then
		assert.NoError(t, err)
	})

	t.Run("ExecStmtHelper", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		query := fmt.Sprintf( /*sql*/ `INSERT INTO conntest_simple (id, val) VALUES (%s, %s)`, conn.FormatPlaceholder(0), conn.FormatPlaceholder(1))
		execFunc, closeStmt, err := sqldb.ExecStmt(ctx, conn, conn, query)
		require.NoError(t, err)
		defer closeStmt() //nolint:errcheck

		// when
		err = execFunc(ctx, 1, "stmt-helper")

		// then
		require.NoError(t, err)
		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "stmt-helper", got.Val)
	})

	t.Run("QueryRowAsStmtHelper", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "stmt-query"})

		query := /*sql*/ `SELECT val FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		queryFunc, closeStmt, err := sqldb.QueryRowAsStmt[string](ctx, conn, refl, conn, query)
		require.NoError(t, err)
		defer closeStmt() //nolint:errcheck

		// when
		val, err := queryFunc(ctx, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, "stmt-query", val)
	})
}
