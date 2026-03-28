package mssqlconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/mssqlconn"
)

type upsertRow struct {
	sqldb.TableName `db:"test_upsert"`

	ID    int    `db:"id,primarykey"`
	Name  string `db:"name"`
	Score int    `db:"score"`
}

func TestUpsertRowStruct(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()
	qb := mssqlconn.QueryBuilder{}

	err = conn.Exec(ctx,
		/*sql*/ `
		IF OBJECT_ID('test_upsert', 'U') IS NOT NULL DROP TABLE test_upsert;
		CREATE TABLE test_upsert (
			id    INT PRIMARY KEY,
			name  NVARCHAR(255) NOT NULL,
			score INT NOT NULL DEFAULT 0
		)`,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Exec(ctx, //nolint:errcheck
			/*sql*/ `DROP TABLE IF EXISTS test_upsert`,
		)
	})

	t.Run("insert new row", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice", Score: 100}
		err := sqldb.UpsertRowStruct(ctx, conn, refl, qb, conn, &row)
		require.NoError(t, err)

		// Verify
		rows := conn.Query(ctx,
			/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = @p1`, 1,
		)
		require.True(t, rows.Next())
		var got upsertRow
		require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
		require.NoError(t, rows.Close())
		assert.Equal(t, upsertRow{ID: 1, Name: "alice", Score: 100}, got)
	})

	t.Run("update existing row", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice_updated", Score: 200}
		err := sqldb.UpsertRowStruct(ctx, conn, refl, qb, conn, &row)
		require.NoError(t, err)

		// Verify updated values
		rows := conn.Query(ctx,
			/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = @p1`, 1,
		)
		require.True(t, rows.Next())
		var got upsertRow
		require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
		require.NoError(t, rows.Close())
		assert.Equal(t, upsertRow{ID: 1, Name: "alice_updated", Score: 200}, got)
	})
}

func TestUpsertRowStructs(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()
	qb := mssqlconn.QueryBuilder{}

	err = conn.Exec(ctx,
		/*sql*/ `
		IF OBJECT_ID('test_upsert', 'U') IS NOT NULL DROP TABLE test_upsert;
		CREATE TABLE test_upsert (
			id    INT PRIMARY KEY,
			name  NVARCHAR(255) NOT NULL,
			score INT NOT NULL DEFAULT 0
		)`,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Exec(ctx, //nolint:errcheck
			/*sql*/ `DROP TABLE IF EXISTS test_upsert`,
		)
	})

	input := []upsertRow{
		{ID: 1, Name: "alice", Score: 10},
		{ID: 2, Name: "bob", Score: 20},
		{ID: 3, Name: "charlie", Score: 30},
	}
	err = sqldb.UpsertRowStructs(ctx, conn, refl, qb, conn, input)
	require.NoError(t, err)

	// Verify all rows
	rows := conn.Query(ctx,
		/*sql*/ `SELECT id, name, score FROM test_upsert ORDER BY id`,
	)
	var got []upsertRow
	for rows.Next() {
		var r upsertRow
		require.NoError(t, rows.Scan(&r.ID, &r.Name, &r.Score))
		got = append(got, r)
	}
	require.NoError(t, rows.Close())
	assert.Equal(t, input, got)
}

func TestInsertUniqueRowStruct(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()
	qb := mssqlconn.QueryBuilder{}

	err = conn.Exec(ctx,
		/*sql*/ `
		IF OBJECT_ID('test_upsert', 'U') IS NOT NULL DROP TABLE test_upsert;
		CREATE TABLE test_upsert (
			id    INT PRIMARY KEY,
			name  NVARCHAR(255) NOT NULL,
			score INT NOT NULL DEFAULT 0
		)`,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Exec(ctx, //nolint:errcheck
			/*sql*/ `DROP TABLE IF EXISTS test_upsert`,
		)
	})

	t.Run("new row returns true", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice", Score: 100}
		inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, refl, qb, conn, &row, "id")
		require.NoError(t, err)
		assert.True(t, inserted)

		// Verify row exists
		rows := conn.Query(ctx,
			/*sql*/ `SELECT name FROM test_upsert WHERE id = @p1`, 1,
		)
		require.True(t, rows.Next())
		var name string
		require.NoError(t, rows.Scan(&name))
		require.NoError(t, rows.Close())
		assert.Equal(t, "alice", name)
	})

	t.Run("existing row returns false", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice_v2", Score: 200}
		inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, refl, qb, conn, &row, "id")
		require.NoError(t, err)
		assert.False(t, inserted)

		// Verify original values unchanged
		rows := conn.Query(ctx,
			/*sql*/ `SELECT name, score FROM test_upsert WHERE id = @p1`, 1,
		)
		require.True(t, rows.Next())
		var name string
		var score int
		require.NoError(t, rows.Scan(&name, &score))
		require.NoError(t, rows.Close())
		assert.Equal(t, "alice", name)
		assert.Equal(t, 100, score)
	})
}
