package mysqlconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/mysqlconn"
)

type upsertRow struct {
	sqldb.TableName `db:"test_upsert"`

	ID    int    `db:"id,primarykey"`
	Name  string `db:"name"`
	Score int    `db:"score"`
}

func createMySQLUpsertTable(t *testing.T, conn sqldb.Connection) {
	t.Helper()
	ctx := t.Context()
	err := conn.Exec(ctx,
		/*sql*/ `DROP TABLE IF EXISTS test_upsert`,
	)
	require.NoError(t, err)
	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_upsert (
			id    INT PRIMARY KEY,
			name  TEXT NOT NULL,
			score INT NOT NULL DEFAULT 0
		)`,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Exec(ctx, //nolint:errcheck
			/*sql*/ `DROP TABLE IF EXISTS test_upsert`,
		)
	})
}

func TestUpsertRowStruct(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	createMySQLUpsertTable(t, conn)

	ctx := t.Context()
	qb := mysqlconn.QueryBuilder{}

	t.Run("insert new row", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice", Score: 100}
		err := sqldb.UpsertRowStruct(ctx, conn, refl, qb, conn, &row)
		require.NoError(t, err)

		// Verify
		rows := conn.Query(ctx,
			/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = ?`, 1,
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
			/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = ?`, 1,
		)
		require.True(t, rows.Next())
		var got upsertRow
		require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
		require.NoError(t, rows.Close())
		assert.Equal(t, upsertRow{ID: 1, Name: "alice_updated", Score: 200}, got)
	})
}

func TestUpsertRowStructs(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	createMySQLUpsertTable(t, conn)

	ctx := t.Context()
	qb := mysqlconn.QueryBuilder{}

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
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	createMySQLUpsertTable(t, conn)

	ctx := t.Context()
	qb := mysqlconn.QueryBuilder{}

	t.Run("insert new row", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice", Score: 100}
		inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, refl, qb, conn, &row, "id")
		require.NoError(t, err)
		assert.True(t, inserted, "new row should be inserted")

		// Verify
		rows := conn.Query(ctx,
			/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = ?`, 1,
		)
		require.True(t, rows.Next())
		var got upsertRow
		require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
		require.NoError(t, rows.Close())
		assert.Equal(t, upsertRow{ID: 1, Name: "alice", Score: 100}, got)
	})

	t.Run("conflict does not insert", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice_updated", Score: 200}
		inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, refl, qb, conn, &row, "id")
		require.NoError(t, err)
		assert.False(t, inserted, "conflicting row should not be inserted")

		// Verify original row is unchanged
		rows := conn.Query(ctx,
			/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = ?`, 1,
		)
		require.True(t, rows.Next())
		var got upsertRow
		require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
		require.NoError(t, rows.Close())
		assert.Equal(t, upsertRow{ID: 1, Name: "alice", Score: 100}, got, "original row should be unchanged")
	})
}
