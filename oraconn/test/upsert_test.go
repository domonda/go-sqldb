package oraconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/oraconn"
)

func TestUpsert(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_upsert (id NUMBER(10) PRIMARY KEY, name VARCHAR2(255), score NUMBER(10))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_upsert`,
	)

	type Row struct {
		sqldb.TableName `db:"test_upsert"`

		ID    int    `db:"id,primarykey"`
		Name  string `db:"name"`
		Score int    `db:"score"`
	}

	// Initial insert via upsert
	row := Row{ID: 1, Name: "alice", Score: 100}
	err = sqldb.UpsertRowStruct(ctx, conn, refl, oraconn.QueryBuilder{}, conn, &row)
	require.NoError(t, err)

	// Verify the inserted row
	rows := conn.Query(ctx,
		/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = :1`, 1,
	)
	require.True(t, rows.Next())
	var got Row
	require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
	assert.Equal(t, Row{ID: 1, Name: "alice", Score: 100}, got)
	require.NoError(t, rows.Close())

	// Update via upsert
	row = Row{ID: 1, Name: "alice-updated", Score: 200}
	err = sqldb.UpsertRowStruct(ctx, conn, refl, oraconn.QueryBuilder{}, conn, &row)
	require.NoError(t, err)

	// Verify the updated row
	rows = conn.Query(ctx,
		/*sql*/ `SELECT id, name, score FROM test_upsert WHERE id = :1`, 1,
	)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
	assert.Equal(t, Row{ID: 1, Name: "alice-updated", Score: 200}, got)
	require.NoError(t, rows.Close())
}

func TestInsertUnique(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_insert_unique (id NUMBER(10) PRIMARY KEY, name VARCHAR2(255))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_insert_unique`,
	)

	type Row struct {
		sqldb.TableName `db:"test_insert_unique"`

		ID   int    `db:"id,primarykey"`
		Name string `db:"name"`
	}

	// First insert should succeed
	row := Row{ID: 1, Name: "alice"}
	err = sqldb.InsertRowStruct(ctx, conn, refl, oraconn.QueryBuilder{}, conn, &row)
	require.NoError(t, err)

	// Duplicate insert should fail with unique violation
	err = sqldb.InsertRowStruct(ctx, conn, refl, oraconn.QueryBuilder{}, conn, &row)
	require.Error(t, err)
	assert.True(t, oraconn.IsUniqueViolation(err), "expected unique violation, got: %v", err)
}
