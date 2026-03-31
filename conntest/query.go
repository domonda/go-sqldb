package conntest

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runQueryTests(t *testing.T, config Config) {
	t.Run("QueryRowAs", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "hello"})

		// when
		query := "SELECT val FROM conntest_simple WHERE id = " + conn.FormatPlaceholder(0)
		val, err := sqldb.QueryRowAs[string](ctx, conn, refl, conn, query, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, "hello", val)
	})

	t.Run("QueryRowStruct", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 42, Val: "found"})

		// when
		got, err := sqldb.QueryRowStruct[simpleRow](ctx, conn, refl, qb, conn, 42)

		// then
		require.NoError(t, err)
		assert.Equal(t, 42, got.ID)
		assert.Equal(t, "found", got.Val)
	})

	t.Run("QueryRowStructOr", func(t *testing.T) {
		t.Run("Found", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
			insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "exists"})

			// when
			got, err := sqldb.QueryRowStructOr(ctx, conn, refl, qb, conn, simpleRow{}, 1)

			// then
			require.NoError(t, err)
			assert.Equal(t, "exists", got.Val)
		})

		t.Run("NotFound", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
			defaultRow := simpleRow{ID: -1, Val: "default"}

			// when
			got, err := sqldb.QueryRowStructOr(ctx, conn, refl, qb, conn, defaultRow, 999)

			// then
			require.NoError(t, err)
			assert.Equal(t, defaultRow, got)
		})
	})

	t.Run("QueryRowsAsSlice", func(t *testing.T) {
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
		got, err := sqldb.QueryRowsAsSlice[simpleRow](ctx, conn, refl, conn, "SELECT * FROM conntest_simple ORDER BY id")

		// then
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "a", got[0].Val)
		assert.Equal(t, "b", got[1].Val)
		assert.Equal(t, "c", got[2].Val)
	})

	t.Run("QueryStructCallback", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		require.NoError(t, sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, []simpleRow{
			{ID: 1, Val: "x"},
			{ID: 2, Val: "y"},
			{ID: 3, Val: "z"},
		}))

		// when
		var collected []simpleRow
		err := sqldb.QueryStructCallback(ctx, conn, refl, conn, func(row simpleRow) error {
			collected = append(collected, row)
			return nil
		}, "SELECT * FROM conntest_simple ORDER BY id")

		// then
		require.NoError(t, err)
		require.Len(t, collected, 3)
		assert.Equal(t, "x", collected[0].Val)
		assert.Equal(t, "z", collected[2].Val)
	})

	t.Run("QueryNoRows", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		query := "SELECT val FROM conntest_simple WHERE id = " + conn.FormatPlaceholder(0)
		_, err := sqldb.QueryRowAs[string](ctx, conn, refl, conn, query, 999)

		// then
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("RawQueryColumns", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "cols"})

		// when
		rows := conn.Query(ctx, "SELECT * FROM conntest_simple")
		defer rows.Close() //nolint:errcheck
		cols, err := rows.Columns()

		// then
		require.NoError(t, err)
		require.Len(t, cols, 2)
	})

	t.Run("RawQueryCloseEarly", func(t *testing.T) {
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

		// when — close before iterating all rows
		rows := conn.Query(ctx, "SELECT * FROM conntest_simple ORDER BY id")
		require.True(t, rows.Next())
		err := rows.Close()

		// then
		assert.NoError(t, err)
	})
}
