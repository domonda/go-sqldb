package conntest

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runExecTests(t *testing.T, config Config) {
	t.Run("Insert", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		err := sqldb.Insert(ctx, conn, qb, conn, "conntest_simple", sqldb.Values{"id": 1, "val": "hello"})

		// then
		require.NoError(t, err)
		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "hello", got.Val)
	})

	t.Run("Update", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "before"})

		// when
		where := "id = " + conn.FormatPlaceholder(0)
		err := sqldb.Update(ctx, conn, qb, conn, "conntest_simple", sqldb.Values{"val": "after"}, where, 1)

		// then
		require.NoError(t, err)
		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "after", got.Val)
	})

	t.Run("InsertRowStruct", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		row := simpleRow{ID: 1, Val: "struct-insert"}

		// when
		err := sqldb.InsertRowStruct(ctx, conn, refl, qb, conn, &row)

		// then
		require.NoError(t, err)
		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "struct-insert", got.Val)
	})

	t.Run("UpdateRowStruct", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
		insertUpsertRow(t, conn, qb, upsertRow{ID: 1, Name: "alice", Score: 10})

		// when
		err := sqldb.UpdateRowStruct(ctx, conn, refl, qb, conn, &upsertRow{ID: 1, Name: "alice_updated", Score: 20})

		// then
		require.NoError(t, err)
		got := queryUpsertRow(t, conn, qb, 1)
		assert.Equal(t, "alice_updated", got.Name)
		assert.Equal(t, 20, got.Score)
	})

	t.Run("DeleteRowStruct", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "to-delete"})

		// when
		err := sqldb.DeleteRowStruct(ctx, conn, refl, qb, conn, &simpleRow{ID: 1})

		// then
		require.NoError(t, err)
		_, err = sqldb.QueryRowByPrimaryKey[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("InsertRowStructs", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		rows := []simpleRow{
			{ID: 1, Val: "a"},
			{ID: 2, Val: "b"},
			{ID: 3, Val: "c"},
		}

		// when
		err := sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, rows)

		// then
		require.NoError(t, err)
		for _, expected := range rows {
			got := querySimpleRow(t, conn, qb, expected.ID)
			assert.Equal(t, expected.Val, got.Val)
		}
	})

	t.Run("ExecRowsAffected", func(t *testing.T) {
		t.Run("DeleteMultiple", func(t *testing.T) {
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

			// when — delete rows with id <= 2
			query := "DELETE FROM conntest_simple WHERE id <= " + conn.FormatPlaceholder(0)
			n, err := sqldb.ExecRowsAffected(ctx, conn, conn, query, 2)

			// then
			require.NoError(t, err)
			assert.Equal(t, int64(2), n)
		})

		t.Run("UpdateMultiple", func(t *testing.T) {
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

			// when — update rows with id >= 2
			query := fmt.Sprintf(
				"UPDATE conntest_simple SET val = %s WHERE id >= %s",
				conn.FormatPlaceholder(0),
				conn.FormatPlaceholder(1),
			)
			n, err := sqldb.ExecRowsAffected(ctx, conn, conn, query, "updated", 2)

			// then
			require.NoError(t, err)
			assert.Equal(t, int64(2), n)
		})

		t.Run("NoMatch", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

			// when — delete from empty table
			query := fmt.Sprintf(
				"DELETE FROM conntest_simple WHERE id = %s",
				conn.FormatPlaceholder(0),
			)
			n, err := sqldb.ExecRowsAffected(ctx, conn, conn, query, 999)

			// then
			require.NoError(t, err)
			assert.Equal(t, int64(0), n)
		})
	})
}
