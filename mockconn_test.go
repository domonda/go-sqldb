package sqldb

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockConn(t *testing.T) {
	t.Run("nil formatter uses StdQueryFormatter", func(t *testing.T) {
		conn := NewMockConn(nil)
		require.NotNil(t, conn)
		assert.Nil(t, conn.QueryFormatter)
		// getQueryFormatter should fall back to StdQueryFormatter
		assert.Equal(t, "?", conn.FormatPlaceholder(0))
	})

	t.Run("custom formatter", func(t *testing.T) {
		formatter := NewQueryFormatter("$")
		conn := NewMockConn(formatter)
		assert.Equal(t, "$1", conn.FormatPlaceholder(0))
	})
}

func TestMockConn_Config(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		conn := NewMockConn(nil)
		cfg := conn.Config()
		assert.Equal(t, "MockConn", cfg.Driver)
	})

	t.Run("custom config", func(t *testing.T) {
		conn := NewMockConn(nil)
		conn.MockConfig = func() *Config {
			return &Config{Driver: "test", Database: "testdb"}
		}
		cfg := conn.Config()
		assert.Equal(t, "test", cfg.Driver)
		assert.Equal(t, "testdb", cfg.Database)
	})
}

func TestMockConn_Ping(t *testing.T) {
	t.Run("nil mock returns context error", func(t *testing.T) {
		conn := NewMockConn(nil)
		err := conn.Ping(t.Context(), 0)
		assert.NoError(t, err)
	})

	t.Run("canceled context", func(t *testing.T) {
		conn := NewMockConn(nil)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := conn.Ping(ctx, 0)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestMockConn_Stats(t *testing.T) {
	t.Run("nil mock returns zero stats", func(t *testing.T) {
		conn := NewMockConn(nil)
		stats := conn.Stats()
		assert.Equal(t, sql.DBStats{}, stats)
	})
}

func TestMockConn_Exec(t *testing.T) {
	t.Run("records exec", func(t *testing.T) {
		conn := NewMockConn(nil)
		err := conn.Exec(t.Context(), "INSERT INTO t VALUES (?)", "a")
		assert.NoError(t, err)
		require.Len(t, conn.Recordings.Execs, 1)
		assert.Equal(t, "INSERT INTO t VALUES (?)", conn.Recordings.Execs[0].Query)
	})

	t.Run("with query log", func(t *testing.T) {
		var buf bytes.Buffer
		conn := NewMockConn(nil).WithQueryLog(&buf)
		err := conn.Exec(t.Context(), "DELETE FROM t WHERE id = ?", int64(1))
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "DELETE FROM t WHERE id = 1")
	})
}

func TestMockConn_Query(t *testing.T) {
	t.Run("no mock returns ErrNoRows", func(t *testing.T) {
		conn := NewMockConn(nil)
		rows := conn.Query(t.Context(), "SELECT 1")
		defer rows.Close()
		assert.False(t, rows.Next())
	})

	t.Run("with mock query results", func(t *testing.T) {
		conn := NewMockConn(NewQueryFormatter("$")).
			WithQueryResult([]string{"id"}, [][]driver.Value{{int64(42)}}, "SELECT id FROM t WHERE name = $1", "test")

		rows := conn.Query(t.Context(), "SELECT id FROM t WHERE name = $1", "test")
		defer rows.Close()
		require.True(t, rows.Next())
		var id int64
		require.NoError(t, rows.Scan(&id))
		assert.Equal(t, int64(42), id)
	})
}

func TestMockConn_Transaction(t *testing.T) {
	t.Run("not in transaction", func(t *testing.T) {
		conn := NewMockConn(nil)
		tx := conn.Transaction()
		assert.False(t, tx.Active())
	})

	t.Run("in transaction", func(t *testing.T) {
		conn := NewMockConn(nil)
		conn.TxID = 1
		tx := conn.Transaction()
		assert.True(t, tx.Active())
	})
}

func TestMockConn_BeginCommitRollback(t *testing.T) {
	t.Run("begin creates transaction clone", func(t *testing.T) {
		conn := NewMockConn(nil)
		txConn, err := conn.Begin(t.Context(), 1, nil)
		require.NoError(t, err)
		assert.True(t, txConn.Transaction().Active())
	})

	t.Run("begin with zero id errors", func(t *testing.T) {
		conn := NewMockConn(nil)
		_, err := conn.Begin(t.Context(), 0, nil)
		assert.Error(t, err)
	})

	t.Run("commit succeeds", func(t *testing.T) {
		conn := NewMockConn(nil)
		assert.NoError(t, conn.Commit())
	})

	t.Run("rollback succeeds", func(t *testing.T) {
		conn := NewMockConn(nil)
		assert.NoError(t, conn.Rollback())
	})
}

func TestMockConn_Listen(t *testing.T) {
	conn := NewMockConn(nil)

	// initially not listening
	assert.False(t, conn.IsListeningOnChannel("test"))

	// listen
	err := conn.ListenOnChannel("test", nil, nil)
	require.NoError(t, err)
	assert.True(t, conn.IsListeningOnChannel("test"))

	// unlisten
	err = conn.UnlistenChannel("test")
	require.NoError(t, err)
	assert.False(t, conn.IsListeningOnChannel("test"))
}

func TestMockConn_Close(t *testing.T) {
	conn := NewMockConn(nil)
	assert.NoError(t, conn.Close())
}

func TestMockConn_DefaultIsolationLevel(t *testing.T) {
	conn := NewMockConn(nil)
	assert.Equal(t, sql.LevelDefault, conn.DefaultIsolationLevel())
}

func TestMockConn_Prepare(t *testing.T) {
	conn := NewMockConn(nil)
	stmt, err := conn.Prepare(t.Context(), "SELECT $1")
	require.NoError(t, err)
	assert.Equal(t, "SELECT $1", stmt.PreparedQuery())
	assert.NoError(t, stmt.Close())
}

func TestMockConn_Clone(t *testing.T) {
	conn := NewMockConn(NewQueryFormatter("$"))
	conn.TxID = 5
	err := conn.ListenOnChannel("ch", nil, nil)
	require.NoError(t, err)

	clone := conn.Clone()
	assert.Equal(t, uint64(5), clone.TxID)
	assert.True(t, clone.IsListeningOnChannel("ch"))

	// modifications to clone don't affect original
	err = clone.UnlistenChannel("ch")
	require.NoError(t, err)
	assert.True(t, conn.IsListeningOnChannel("ch"))
	assert.False(t, clone.IsListeningOnChannel("ch"))
}

func TestMockConn_WithNormalizeQuery(t *testing.T) {
	conn := NewMockConn(nil)
	assert.Nil(t, conn.NormalizeQuery)

	conn.WithNormalizeQuery(NoChangeNormalizeQuery)
	assert.NotNil(t, conn.NormalizeQuery)
}

func TestMockConn_MaxArgs(t *testing.T) {
	t.Run("default delegates to formatter", func(t *testing.T) {
		conn := NewMockConn(nil)
		// StdQueryFormatter{}.MaxArgs() returns 65535
		assert.Equal(t, StdQueryFormatter{}.MaxArgs(), conn.MaxArgs())
	})

	t.Run("MockMaxArgs overrides", func(t *testing.T) {
		conn := NewMockConn(nil)
		conn.MockMaxArgs = 100
		assert.Equal(t, 100, conn.MaxArgs())
	})
}
