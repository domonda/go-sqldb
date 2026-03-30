package conntest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runReturningTests(t *testing.T, config Config) {
	rqb, ok := config.QueryBuilder.(sqldb.ReturningQueryBuilder)
	if !ok {
		t.Skip("QueryBuilder does not implement ReturningQueryBuilder")
	}
	if config.DDL.CreateReturningTable == "" {
		t.Skip("no DDL provided for returning table")
	}

	t.Run("InsertReturning", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateReturningTable, "conntest_returning")

		// when
		var id int
		var name string
		var score int
		err := sqldb.InsertReturning(ctx, conn, refl, rqb, conn,
			"conntest_returning",
			sqldb.Values{"name": "bob", "score": 42},
			"id, name, score",
		).Scan(&id, &name, &score)

		// then
		require.NoError(t, err)
		assert.Greater(t, id, 0)
		assert.Equal(t, "bob", name)
		assert.Equal(t, 42, score)
	})

	t.Run("InsertReturningDefault", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateReturningTable, "conntest_returning")

		// when
		var id int
		var score int
		err := sqldb.InsertReturning(ctx, conn, refl, rqb, conn,
			"conntest_returning",
			sqldb.Values{"name": "charlie"},
			"id, score",
		).Scan(&id, &score)

		// then
		require.NoError(t, err)
		assert.Greater(t, id, 0)
		assert.Equal(t, 0, score, "score should be DB default 0")
	})

	t.Run("UpdateReturningRow", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateReturningTable, "conntest_returning")

		// Insert a seed row using InsertReturning to get the auto-generated id
		var seedID int
		err := sqldb.InsertReturning(ctx, conn, refl, rqb, conn,
			"conntest_returning",
			sqldb.Values{"name": "alice", "score": 50},
			"id",
		).Scan(&seedID)
		require.NoError(t, err)
		_ = qb // satisfy linter

		// when
		var id int
		var name string
		var score int
		where := "id = " + conn.FormatPlaceholder(0)
		err = sqldb.UpdateReturningRow(ctx, conn, refl, rqb, conn,
			"conntest_returning",
			sqldb.Values{"score": 99},
			"id, name, score",
			where, seedID,
		).Scan(&id, &name, &score)

		// then
		require.NoError(t, err)
		assert.Equal(t, seedID, id)
		assert.Equal(t, "alice", name)
		assert.Equal(t, 99, score)
	})

	t.Run("UpdateReturningRows", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateReturningTable, "conntest_returning")

		// Insert multiple seed rows
		for _, name := range []string{"alice", "bob", "charlie"} {
			err := sqldb.InsertReturning(ctx, conn, refl, rqb, conn,
				"conntest_returning",
				sqldb.Values{"name": name, "score": 10},
				"id",
			).Scan(new(int))
			require.NoError(t, err)
		}

		// when — update all rows and return them
		rows := sqldb.UpdateReturningRows(ctx, conn, rqb, conn,
			"conntest_returning",
			sqldb.Values{"score": 0},
			"id, name, score",
			"score > 0",
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

		// then
		require.Len(t, got, 3)
		for _, r := range got {
			assert.Equal(t, 0, r.Score, "score should be updated to 0")
		}
	})
}
