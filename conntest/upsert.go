package conntest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runUpsertTests(t *testing.T, config Config) {
	uqb, ok := config.QueryBuilder.(sqldb.UpsertQueryBuilder)
	if !ok {
		t.Skip("QueryBuilder does not implement UpsertQueryBuilder")
	}

	t.Run("UpsertRowStruct", func(t *testing.T) {
		t.Run("InsertNew", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
			row := upsertRow{ID: 1, Name: "alice", Score: 100}

			// when
			err := sqldb.UpsertRowStruct(ctx, conn, refl, uqb, conn, &row)

			// then
			require.NoError(t, err)
			got := queryUpsertRow(t, conn, qb, 1)
			assert.Equal(t, "alice", got.Name)
			assert.Equal(t, 100, got.Score)
		})

		t.Run("UpdateExisting", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
			require.NoError(t, sqldb.UpsertRowStruct(ctx, conn, refl, uqb, conn, &upsertRow{ID: 1, Name: "alice", Score: 100}))

			// when
			err := sqldb.UpsertRowStruct(ctx, conn, refl, uqb, conn, &upsertRow{ID: 1, Name: "alice_updated", Score: 200})

			// then
			require.NoError(t, err)
			got := queryUpsertRow(t, conn, qb, 1)
			assert.Equal(t, "alice_updated", got.Name)
			assert.Equal(t, 200, got.Score)
		})
	})

	t.Run("UpsertRowStructs", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
		input := []upsertRow{
			{ID: 1, Name: "alice", Score: 10},
			{ID: 2, Name: "bob", Score: 20},
			{ID: 3, Name: "charlie", Score: 30},
		}

		// when
		err := sqldb.UpsertRowStructs(ctx, conn, refl, uqb, conn, input)

		// then
		require.NoError(t, err)
		got, err := sqldb.QueryRowsAsSlice[upsertRow](ctx, conn, refl, conn, "SELECT * FROM conntest_upsert ORDER BY id")
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "alice", got[0].Name)
		assert.Equal(t, "bob", got[1].Name)
		assert.Equal(t, "charlie", got[2].Name)
	})

	t.Run("InsertUnique", func(t *testing.T) {
		t.Run("NewRow", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")

			// when
			inserted, err := sqldb.InsertUnique(ctx, conn, uqb, conn, "conntest_upsert",
				sqldb.Values{"id": 1, "name": "alice", "score": 100}, "id")

			// then
			require.NoError(t, err)
			assert.True(t, inserted)
			got := queryUpsertRow(t, conn, qb, 1)
			assert.Equal(t, "alice", got.Name)
		})

		t.Run("Conflict", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
			insertUpsertRow(t, conn, qb, upsertRow{ID: 1, Name: "alice", Score: 100})

			// when
			inserted, err := sqldb.InsertUnique(ctx, conn, uqb, conn, "conntest_upsert",
				sqldb.Values{"id": 1, "name": "alice_v2", "score": 200}, "id")

			// then
			require.NoError(t, err)
			assert.False(t, inserted)
			got := queryUpsertRow(t, conn, qb, 1)
			assert.Equal(t, "alice", got.Name, "original row should be unchanged")
			assert.Equal(t, 100, got.Score, "original score should be unchanged")
		})
	})

	t.Run("InsertUniqueRowStruct", func(t *testing.T) {
		t.Run("NewRow", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
			row := upsertRow{ID: 1, Name: "alice", Score: 100}

			// when
			inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, refl, uqb, conn, &row, "id")

			// then
			require.NoError(t, err)
			assert.True(t, inserted)
			got := queryUpsertRow(t, conn, qb, 1)
			assert.Equal(t, "alice", got.Name)
		})

		t.Run("Conflict", func(t *testing.T) {
			// given
			conn := config.NewConn(t)
			ctx := t.Context()
			qb := config.QueryBuilder
			setupTable(t, conn, config.DDL.CreateUpsertTable, "conntest_upsert")
			insertUpsertRow(t, conn, qb, upsertRow{ID: 1, Name: "alice", Score: 100})

			// when
			row := upsertRow{ID: 1, Name: "alice_v2", Score: 200}
			inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, refl, uqb, conn, &row, "id")

			// then
			require.NoError(t, err)
			assert.False(t, inserted)
			got := queryUpsertRow(t, conn, qb, 1)
			assert.Equal(t, "alice", got.Name, "original row should be unchanged")
			assert.Equal(t, 100, got.Score, "original score should be unchanged")
		})
	})
}
