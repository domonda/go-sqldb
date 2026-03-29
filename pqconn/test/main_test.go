package pqconn

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/pqconn"
)

var (
	postgresUser     = envOrDefault("POSTGRES_USER", "testuser")
	postgresPassword = envOrDefault("POSTGRES_PASSWORD", "testpassword")
	postgresHost     = envOrDefault("POSTGRES_HOST", "localhost")
	postgresPort     = envOrDefault("POSTGRES_PORT", "5433")
	dbName           = envOrDefault("POSTGRES_DB", "testdb")
)

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func dockerComposeUp() error {
	return exec.Command("docker", "compose", "up", "-d").Run()
}

func dropSchemaTables() error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(pqconn.DropAllInCurrentSchemaQuery)
	return err
}

func TestMain(m *testing.M) {
	if os.Getenv("CI") == "" {
		err := dockerComposeUp()
		if err != nil {
			log.Fatalf("Failed to start Docker Compose: %v", err)
		}
	}

	err := dropSchemaTables()
	if err != nil {
		log.Fatalf("Failed to drop all user data before tests: %v", err)
	}

	m.Run()
}

func TestConfig(t *testing.T) {
	conn := connectPQ(t)
	cfg := conn.Config()
	require.NotNil(t, cfg)
	assert.Equal(t, pqconn.Driver, cfg.Driver)
	assert.Equal(t, dbName, cfg.Database)
}

func TestPing(t *testing.T) {
	conn := connectPQ(t)
	err := conn.Ping(t.Context(), 5*time.Second)
	assert.NoError(t, err)
}

func TestStats(t *testing.T) {
	conn := connectPQ(t)
	// Stats() should return without panic; exact values depend on pool state
	_ = conn.Stats()
}

func TestDefaultIsolationLevel(t *testing.T) {
	conn := connectPQ(t)
	assert.Equal(t, sql.LevelReadCommitted, conn.DefaultIsolationLevel())
}

func TestTransactionState(t *testing.T) {
	conn := connectPQ(t)

	t.Run("not in transaction", func(t *testing.T) {
		tx := conn.Transaction()
		assert.False(t, tx.Active())
	})

	t.Run("in transaction", func(t *testing.T) {
		txConn, err := conn.Begin(t.Context(), 1, nil)
		require.NoError(t, err)
		defer txConn.Rollback() //nolint:errcheck

		tx := txConn.Transaction()
		assert.True(t, tx.Active())
	})
}

func TestExecRowsAffected(t *testing.T) {
	conn := connectPQ(t)
	ctx := t.Context()

	err := conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS _test_rows_affected (id SERIAL PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Exec(ctx, `DROP TABLE IF EXISTS _test_rows_affected`) }) //nolint:errcheck

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO _test_rows_affected (val) VALUES ($1), ($2), ($3)`, "a", "b", "c",
	)
	require.NoError(t, err)

	n, err := conn.ExecRowsAffected(ctx,
		/*sql*/ `DELETE FROM _test_rows_affected WHERE val IN ($1, $2)`, "a", "b",
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestPrepare(t *testing.T) {
	conn := connectPQ(t)
	ctx := t.Context()

	err := conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS _test_prepare (id SERIAL PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Exec(ctx, `DROP TABLE IF EXISTS _test_prepare`) }) //nolint:errcheck

	stmt, err := conn.Prepare(ctx,
		/*sql*/ `INSERT INTO _test_prepare (val) VALUES ($1)`,
	)
	require.NoError(t, err)
	defer stmt.Close() //nolint:errcheck

	err = stmt.Exec(ctx, "prepared-value")
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM _test_prepare LIMIT 1`,
	)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "prepared-value", val)
	require.NoError(t, rows.Close())

	require.NoError(t, stmt.Close())
}

func TestListenUnlisten(t *testing.T) {
	conn := connectPQ(t)

	listenerConn, ok := conn.(sqldb.ListenerConnection)
	if !ok {
		t.Skip("connection does not implement ListenerConnection")
	}

	assert.False(t, listenerConn.IsListeningOnChannel("test_channel"))

	err := listenerConn.ListenOnChannel("test_channel", func(channel, payload string) {}, func(channel string) {})
	if err != nil {
		t.Skipf("ListenOnChannel failed (listener may need separate auth): %v", err)
	}
	assert.True(t, listenerConn.IsListeningOnChannel("test_channel"))

	err = listenerConn.UnlistenChannel("test_channel")
	require.NoError(t, err)

	// Give a moment for unlisten to propagate
	time.Sleep(100 * time.Millisecond)
	assert.False(t, listenerConn.IsListeningOnChannel("test_channel"))
}

func TestDatabase(t *testing.T) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	t.Run("ping succeeds", func(t *testing.T) {
		if err := db.PingContext(t.Context()); err != nil {
			t.Fatalf("Ping failed: %v", err)
		}
	})

	t.Run("simple scalar query", func(t *testing.T) {
		var result int
		if err := db.QueryRowContext(t.Context(), "SELECT 1").Scan(&result); err != nil {
			t.Fatalf("QueryRow failed: %v", err)
		}
		if result != 1 {
			t.Errorf("SELECT 1 returned %d, want 1", result)
		}
	})

	t.Run("create table insert and select", func(t *testing.T) {
		createSQL := /*sql*/ `
			CREATE TABLE IF NOT EXISTS _test_db (
				id   SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			)`
		_, err := db.ExecContext(t.Context(), createSQL)
		if err != nil {
			t.Fatalf("CREATE TABLE failed: %v", err)
		}
		t.Cleanup(func() {
			dropSQL := /*sql*/ `DROP TABLE IF EXISTS _test_db`
			db.ExecContext(t.Context(), dropSQL) //nolint:errcheck
		})

		insertSQL := /*sql*/ `INSERT INTO _test_db (name) VALUES ($1)`
		_, err = db.ExecContext(t.Context(), insertSQL, "hello")
		if err != nil {
			t.Fatalf("INSERT failed: %v", err)
		}

		selectSQL := /*sql*/ `SELECT name FROM _test_db LIMIT 1`
		var name string
		if err := db.QueryRowContext(t.Context(), selectSQL).Scan(&name); err != nil {
			t.Fatalf("SELECT failed: %v", err)
		}
		if name != "hello" {
			t.Errorf("name = %q, want %q", name, "hello")
		}
	})

	t.Run("transaction commit", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		var n int
		if err := tx.QueryRowContext(t.Context(), "SELECT 42").Scan(&n); err != nil {
			tx.Rollback()
			t.Fatalf("QueryRow in tx failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
		if n != 42 {
			t.Errorf("SELECT 42 returned %d, want 42", n)
		}
	})

	t.Run("transaction rollback", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}
	})
}
