package conntest

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runQueryBuilderTests(t *testing.T, config Config) {
	t.Run("InsertAndQueryByPrimaryKey", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		row := simpleRow{ID: 1, Val: "round-trip"}

		// when
		err := sqldb.InsertRowStruct(ctx, conn, refl, qb, conn, &row)
		require.NoError(t, err)

		got, err := sqldb.QueryRowByPrimaryKey[simpleRow](ctx, conn, refl, qb, conn, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, row.ID, got.ID)
		assert.Equal(t, row.Val, got.Val)
	})

	t.Run("InsertRowStructs", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		rows := []simpleRow{
			{ID: 1, Val: "first"},
			{ID: 2, Val: "second"},
			{ID: 3, Val: "third"},
		}

		// when
		err := sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, rows)
		require.NoError(t, err)

		got, err := sqldb.QueryRowsAsSlice[simpleRow](ctx, conn, refl, conn, "SELECT * FROM conntest_simple ORDER BY id")

		// then
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "first", got[0].Val)
		assert.Equal(t, "second", got[1].Val)
		assert.Equal(t, "third", got[2].Val)
	})

	t.Run("UpdateRowStruct", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
		insertUpsertRow(t, conn, qb, upsertRow{ID: 1, Name: "before", Score: 10})

		// when
		err := sqldb.UpdateRowStruct(ctx, conn, refl, qb, conn, &upsertRow{ID: 1, Name: "after", Score: 99})
		require.NoError(t, err)

		got := queryUpsertRow(t, conn, qb, 1)

		// then
		assert.Equal(t, "after", got.Name)
		assert.Equal(t, 99, got.Score)
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
		require.NoError(t, err)

		_, err = sqldb.QueryRowByPrimaryKey[simpleRow](ctx, conn, refl, qb, conn, 1)

		// then
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("InsertValues", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		err := sqldb.Insert(ctx, conn, qb, conn, "conntest_simple", sqldb.Values{"id": 1, "val": "values-insert"})
		require.NoError(t, err)

		got := querySimpleRow(t, conn, qb, 1)

		// then
		assert.Equal(t, "values-insert", got.Val)
	})

	t.Run("UpdateValues", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "before"})

		// when
		where := "id = " + conn.FormatPlaceholder(0)
		err := sqldb.Update(ctx, conn, qb, conn, "conntest_simple", sqldb.Values{"val": "after"}, where, 1)
		require.NoError(t, err)

		got := querySimpleRow(t, conn, qb, 1)

		// then
		assert.Equal(t, "after", got.Val)
	})
}
