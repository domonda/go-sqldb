package conntest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runBatchTests(t *testing.T, config Config) {
	t.Run("UpdateRowStructs", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
		require.NoError(t, sqldb.InsertRowStructs(ctx, conn, refl, qb, conn, []upsertRow{
			{ID: 1, Name: "alice", Score: 10},
			{ID: 2, Name: "bob", Score: 20},
			{ID: 3, Name: "charlie", Score: 30},
		}))

		// when
		err := sqldb.UpdateRowStructs(ctx, conn, refl, qb, conn, []upsertRow{
			{ID: 1, Name: "alice_updated", Score: 11},
			{ID: 2, Name: "bob_updated", Score: 22},
			{ID: 3, Name: "charlie_updated", Score: 33},
		})

		// then
		require.NoError(t, err)
		assert.Equal(t, "alice_updated", queryUpsertRow(t, conn, qb, 1).Name)
		assert.Equal(t, 22, queryUpsertRow(t, conn, qb, 2).Score)
		assert.Equal(t, "charlie_updated", queryUpsertRow(t, conn, qb, 3).Name)
	})

	t.Run("UpdateRowStructs/Empty", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when — empty slice should be a no-op
		err := sqldb.UpdateRowStructs(ctx(t), conn, refl, config.QueryBuilder, conn, []upsertRow{})

		// then
		require.NoError(t, err)
	})

	t.Run("UpdateRowStructs/Single", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
		insertUpsertRow(t, conn, qb, upsertRow{ID: 1, Name: "alice", Score: 10})

		// when — single element takes the non-transaction path
		err := sqldb.UpdateRowStructs(ctx, conn, refl, qb, conn, []upsertRow{
			{ID: 1, Name: "alice_single", Score: 99},
		})

		// then
		require.NoError(t, err)
		got := queryUpsertRow(t, conn, qb, 1)
		assert.Equal(t, "alice_single", got.Name)
		assert.Equal(t, 99, got.Score)
	})

	t.Run("DeleteRowStructs", func(t *testing.T) {
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

		// when — delete first two rows
		err := sqldb.DeleteRowStructs(ctx, conn, refl, qb, conn, []simpleRow{
			{ID: 1},
			{ID: 2},
		})

		// then — only the third row remains
		require.NoError(t, err)
		_, err = sqldb.QueryRowStruct[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		_, err = sqldb.QueryRowStruct[simpleRow](ctx, conn, refl, qb, conn, 2)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		got := querySimpleRow(t, conn, qb, 3)
		assert.Equal(t, "c", got.Val)
	})

	t.Run("DeleteRowStructs/Empty", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when — empty slice should be a no-op
		err := sqldb.DeleteRowStructs(ctx(t), conn, refl, config.QueryBuilder, conn, []simpleRow{})

		// then
		require.NoError(t, err)
	})

	t.Run("DeleteRowStructs/Single", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")
		insertSimpleRow(t, conn, qb, simpleRow{ID: 1, Val: "only"})

		// when — single element takes the non-transaction path
		err := sqldb.DeleteRowStructs(ctx, conn, refl, qb, conn, []simpleRow{{ID: 1}})

		// then
		require.NoError(t, err)
		_, err = sqldb.QueryRowStruct[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("DeleteRowStructs/NotFound", func(t *testing.T) {
		// given — empty table, no rows to match
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		err := sqldb.DeleteRowStructs(ctx, conn, refl, qb, conn, []simpleRow{{ID: 999}})

		// then
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("ExecRowsAffectedStmt", func(t *testing.T) {
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

		query := "DELETE FROM conntest_simple WHERE id = " + conn.FormatPlaceholder(0)
		execFunc, closeStmt, err := sqldb.ExecRowsAffectedStmt(ctx, conn, conn, query)
		require.NoError(t, err)
		defer closeStmt() //nolint:errcheck

		// when — delete rows one by one
		n1, err1 := execFunc(ctx, 1)
		n2, err2 := execFunc(ctx, 2)
		n3, err3 := execFunc(ctx, 999) // no match

		// then
		require.NoError(t, err1)
		assert.Equal(t, int64(1), n1)
		require.NoError(t, err2)
		assert.Equal(t, int64(1), n2)
		require.NoError(t, err3)
		assert.Equal(t, int64(0), n3)
	})

	t.Run("ExecRowsAffectedStmt/UpdateMultiple", func(t *testing.T) {
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

		query := fmt.Sprintf(
			"UPDATE conntest_simple SET val = %s WHERE id <= %s",
			conn.FormatPlaceholder(0),
			conn.FormatPlaceholder(1),
		)
		execFunc, closeStmt, err := sqldb.ExecRowsAffectedStmt(ctx, conn, conn, query)
		require.NoError(t, err)
		defer closeStmt() //nolint:errcheck

		// when
		n, err := execFunc(ctx, "updated", 2)

		// then
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)
	})
}

// ctx is a convenience helper for tests that only need a context.
func ctx(t *testing.T) context.Context {
	t.Helper()
	return t.Context()
}
