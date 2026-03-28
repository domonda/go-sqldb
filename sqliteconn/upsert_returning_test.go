package sqliteconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

var upsertRefl = sqldb.NewTaggedStructReflector()

type upsertRow struct {
	sqldb.TableName `db:"test_upsert"`

	ID    int    `db:"id,primarykey"`
	Name  string `db:"name"`
	Score int    `db:"score"`
}

func connectSQLite(t *testing.T) sqldb.Connection {
	t.Helper()
	config := &sqldb.ConnConfig{
		Driver:   "sqlite",
		Host:     "localhost",
		Database: ":memory:",
	}
	conn, err := Connect(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func createUpsertTable(t *testing.T, conn sqldb.Connection) {
	t.Helper()
	err := conn.Exec(t.Context(),
		/*sql*/ `CREATE TABLE test_upsert (
			id    INTEGER PRIMARY KEY,
			name  TEXT NOT NULL,
			score INTEGER NOT NULL DEFAULT 0
		)`,
	)
	require.NoError(t, err)
}

func createReturningTable(t *testing.T, conn sqldb.Connection) {
	t.Helper()
	err := conn.Exec(t.Context(),
		/*sql*/ `CREATE TABLE test_returning (
			id    INTEGER PRIMARY KEY AUTOINCREMENT,
			name  TEXT NOT NULL,
			score INTEGER NOT NULL DEFAULT 0
		)`,
	)
	require.NoError(t, err)
}

func TestUpsertRowStruct(t *testing.T) {
	conn := connectSQLite(t)
	createUpsertTable(t, conn)

	ctx := t.Context()
	qb := QueryBuilder{}

	t.Run("insert new row", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice", Score: 100}
		err := sqldb.UpsertRowStruct(ctx, conn, upsertRefl, qb, conn, &row)
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
		err := sqldb.UpsertRowStruct(ctx, conn, upsertRefl, qb, conn, &row)
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
	conn := connectSQLite(t)
	createUpsertTable(t, conn)

	ctx := t.Context()
	qb := QueryBuilder{}

	input := []upsertRow{
		{ID: 1, Name: "alice", Score: 10},
		{ID: 2, Name: "bob", Score: 20},
		{ID: 3, Name: "charlie", Score: 30},
	}
	err := sqldb.UpsertRowStructs(ctx, conn, upsertRefl, qb, conn, input)
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
	conn := connectSQLite(t)
	createUpsertTable(t, conn)

	ctx := t.Context()
	qb := QueryBuilder{}

	t.Run("new row returns true", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice", Score: 100}
		inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, upsertRefl, qb, conn, &row, "id")
		require.NoError(t, err)
		assert.True(t, inserted)

		// Verify row exists
		rows := conn.Query(ctx,
			/*sql*/ `SELECT name FROM test_upsert WHERE id = ?`, 1,
		)
		require.True(t, rows.Next())
		var name string
		require.NoError(t, rows.Scan(&name))
		require.NoError(t, rows.Close())
		assert.Equal(t, "alice", name)
	})

	t.Run("existing row returns false", func(t *testing.T) {
		row := upsertRow{ID: 1, Name: "alice_v2", Score: 200}
		inserted, err := sqldb.InsertUniqueRowStruct(ctx, conn, upsertRefl, qb, conn, &row, "id")
		require.NoError(t, err)
		assert.False(t, inserted)

		// Verify original values unchanged
		rows := conn.Query(ctx,
			/*sql*/ `SELECT name, score FROM test_upsert WHERE id = ?`, 1,
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

func TestInsertReturning(t *testing.T) {
	conn := connectSQLite(t)
	createReturningTable(t, conn)

	ctx := t.Context()
	qb := QueryBuilder{}

	t.Run("returns inserted values", func(t *testing.T) {
		var id int
		var name string
		var score int
		err := sqldb.InsertReturning(ctx, conn, upsertRefl, qb, conn,
			"test_returning",
			sqldb.Values{"name": "bob", "score": 42},
			"id, name, score",
		).Scan(&id, &name, &score)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
		assert.Equal(t, "bob", name)
		assert.Equal(t, 42, score)
	})

	t.Run("returns database default", func(t *testing.T) {
		var id int
		var score int
		err := sqldb.InsertReturning(ctx, conn, upsertRefl, qb, conn,
			"test_returning",
			sqldb.Values{"name": "charlie"},
			"id, score",
		).Scan(&id, &score)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
		assert.Equal(t, 0, score, "score should be DB default 0")
	})
}

func TestUpdateReturningRow(t *testing.T) {
	conn := connectSQLite(t)
	createReturningTable(t, conn)

	ctx := t.Context()
	qb := QueryBuilder{}

	// Insert a row to update
	err := conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_returning (id, name, score) VALUES (?, ?, ?)`, 1, "alice", 50,
	)
	require.NoError(t, err)

	var id int
	var name string
	var score int
	err = sqldb.UpdateReturningRow(ctx, conn, upsertRefl, qb, conn,
		"test_returning",
		sqldb.Values{"score": 99},
		"id, name, score",
		"id = ?", 1,
	).Scan(&id, &name, &score)
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.Equal(t, "alice", name)
	assert.Equal(t, 99, score)
}

func TestUpdateReturningRows(t *testing.T) {
	conn := connectSQLite(t)
	createReturningTable(t, conn)

	ctx := t.Context()
	qb := QueryBuilder{}

	// Insert multiple rows
	for _, r := range []struct {
		id    int
		name  string
		score int
	}{
		{1, "alice", 10},
		{2, "bob", 20},
		{3, "charlie", 30},
	} {
		err := conn.Exec(ctx,
			/*sql*/ `INSERT INTO test_returning (id, name, score) VALUES (?, ?, ?)`, r.id, r.name, r.score,
		)
		require.NoError(t, err)
	}

	// Update rows with score > 15 and return them
	rows := sqldb.UpdateReturningRows(ctx, conn, qb, conn,
		"test_returning",
		sqldb.Values{"score": 0},
		"id, name, score",
		"score > ?", 15,
	)
	type result struct {
		ID    int
		Name  string
		Score int
	}
	var got []result
	for rows.Next() {
		var r result
		require.NoError(t, rows.Scan(&r.ID, &r.Name, &r.Score))
		got = append(got, r)
	}
	require.NoError(t, rows.Close())
	require.Len(t, got, 2)
	for _, r := range got {
		assert.Equal(t, 0, r.Score, "score should be updated to 0")
		assert.Contains(t, []string{"bob", "charlie"}, r.Name)
	}
}
