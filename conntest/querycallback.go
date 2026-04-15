package conntest

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runQueryCallbackTests(t *testing.T, config Config) {
	t.Run("QueryRowAsMap", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "mapped"})

		// when
		query := /*sql*/ `SELECT id, val FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		m, err := sqldb.QueryRowAsMap[string, any](ctx, conn, conn, query, 1)

		// then
		require.NoError(t, err)
		require.Len(t, m, 2)
		assert.Contains(t, m, "id")
		assert.Contains(t, m, "val")
	})

	t.Run("QueryRowAsMap/NoRows", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		query := /*sql*/ `SELECT id, val FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		_, err := sqldb.QueryRowAsMap[string, any](ctx, conn, conn, query, 999)

		// then
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("QueryCallback/ScalarArgs", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		require.NoError(t, sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, []simpleRow{
			{ID: 1, Val: "a"},
			{ID: 2, Val: "b"},
			{ID: 3, Val: "c"},
		}))

		// when
		var collected []string
		err := sqldb.QueryCallback(ctx, conn, refl, conn,
			func(id int, val string) {
				collected = append(collected, val)
			},
			/*sql*/ `SELECT id, val FROM conntest_simple ORDER BY id`,
		)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, collected)
	})

	t.Run("QueryCallback/StructArg", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		require.NoError(t, sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, []simpleRow{
			{ID: 1, Val: "x"},
			{ID: 2, Val: "y"},
		}))

		// when
		var collected []simpleRow
		err := sqldb.QueryCallback(ctx, conn, refl, conn,
			func(row simpleRow) {
				collected = append(collected, row)
			},
			/*sql*/ `SELECT * FROM conntest_simple ORDER BY id`,
		)

		// then
		require.NoError(t, err)
		require.Len(t, collected, 2)
		assert.Equal(t, "x", collected[0].Val)
		assert.Equal(t, "y", collected[1].Val)
	})

	t.Run("QueryCallback/ErrorReturn", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		require.NoError(t, sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, []simpleRow{
			{ID: 1, Val: "a"},
			{ID: 2, Val: "b"},
			{ID: 3, Val: "c"},
		}))

		// when — callback returns error on second row
		stopErr := errors.New("stop")
		var count int
		err := sqldb.QueryCallback(ctx, conn, refl, conn,
			func(row simpleRow) error {
				count++
				if count == 2 {
					return stopErr
				}
				return nil
			},
			/*sql*/ `SELECT * FROM conntest_simple ORDER BY id`,
		)

		// then — should propagate error and stop iteration
		assert.ErrorIs(t, err, stopErr)
		assert.Equal(t, 2, count)
	})

	t.Run("QueryCallback/NoRows", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		var called bool
		err := sqldb.QueryCallback(ctx, conn, refl, conn,
			func(id int, val string) {
				called = true
			},
			/*sql*/ `SELECT id, val FROM conntest_simple`,
		)

		// then — no error, callback never called
		require.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("QueryRowAsOr/Found", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "real"})

		// when
		query := /*sql*/ `SELECT val FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		val, err := sqldb.QueryRowAsOr(ctx, conn, refl, conn, "default", query, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, "real", val)
	})

	t.Run("QueryRowAsOr/NotFound", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		query := /*sql*/ `SELECT val FROM conntest_simple WHERE id = ` + conn.FormatPlaceholder(0)
		val, err := sqldb.QueryRowAsOr(ctx, conn, refl, conn, "default", query, 999)

		// then
		require.NoError(t, err)
		assert.Equal(t, "default", val)
	})
}
